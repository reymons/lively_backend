package rtmp

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	rtmplib "github.com/reymons/rtmp-go"

	"lively/container/flv"
	"lively/core/media"
	"lively/core/model"
	"lively/core/service"
)

var (
	errInvalidUserData   = errors.New("userData is not a session")
	errInvalidVideoCodec = errors.New("invalid video codec")
	errInvalidAudioCodec = errors.New("invalid audio codec")
	errInvalidFLVPacket  = errors.New("invalid flv packet")
)

const (
	naluHdrSize    = 5
	naluTypeIDR    = 5
	naluTypeNonIDR = 1
)

type rtmpSession struct {
	conn    *rtmplib.Conn
	userID  uint64
	pub     media.Publisher
	ctx     context.Context
	naluHdr [naluHdrSize]byte
}

type Transport struct {
	ln            rtmplib.Listener
	sender        media.Sender
	skService     service.StreamKey
	streamService service.Stream
	discardBuf    []byte // used for discarding NALU
}

func NewTransport(sender media.Sender, skService service.StreamKey, streamService service.Stream) *Transport {
	return &Transport{
		sender:        sender,
		skService:     skService,
		streamService: streamService,
		discardBuf:    make([]byte, 16*1024),
	}
}

func (t *Transport) sendSeqHeader(typ uint8, r io.Reader, toRead uint32, session *rtmpSession) error {
	sharedBuf := session.pub.AcquireBuffer()
	defer sharedBuf.Release()

	n, err := sharedBuf.ReadN(r, int(toRead))
	if err != nil {
		return fmt.Errorf("read seq header: %w", err)
	}
	if n != int(toRead) {
		return fmt.Errorf("seq header was not fully read")
	}

	mediaMesg := media.Message{
		Type:   typ,
		Length: uint32(sharedBuf.Len()),
		Data:   sharedBuf,
	}
	session.pub.SendMessage(&mediaMesg)
	return nil
}

func (t *Transport) discardNALU(r io.Reader, naluLen uint32) error {
	buf := t.discardBuf
	var discarded uint32

	for discarded < naluLen {
		toRead := uint32(len(buf))
		if remaining := naluLen - discarded; remaining < toRead {
			toRead = remaining
		}

		n, err := r.Read(buf[:toRead])
		if n > 0 {
			discarded += uint32(n)
		}

		if err != nil {
			if err == io.EOF && discarded < naluLen {
				return io.ErrUnexpectedEOF
			}
			return err
		}
	}

	return nil
}

func (t *Transport) sendVideoData(mesg *rtmplib.VideoMessage, session *rtmpSession) (bool, error) {
	naluHdr := session.naluHdr[:]
	if _, err := io.ReadFull(mesg.Data, naluHdr); err != nil {
		if err == io.EOF {
			return true, nil
		}
		return false, io.ErrUnexpectedEOF
	}

	naluLen := binary.BigEndian.Uint32(naluHdr) - 1 // -1 since we've already read nalu type which's a part of nalu length
	naluType := uint8(naluHdr[4] & 0b00011111)

	if naluType != naluTypeNonIDR && naluType != naluTypeIDR {
		if err := t.discardNALU(mesg.Data, naluLen); err != nil {
			return false, fmt.Errorf("discard NALU: %w", err)
		}
		return false, nil
	}

	sharedBuf := session.pub.AcquireBuffer()
	if _, err := sharedBuf.Write(naluHdr); err != nil {
		sharedBuf.Release()
		return false, fmt.Errorf("write NALU header: %w", err)
	}

	firstMesg := true
	remaining := int(naluLen)

	for remaining > 0 {
		if sharedBuf == nil {
			sharedBuf = session.pub.AcquireBuffer()
		}

		n, err := sharedBuf.ReadN(mesg.Data, remaining)
		if err != nil {
			sharedBuf.Release()
			return false, fmt.Errorf("ReadN: %w", err)
		}
		remaining -= n

		mesg := media.Message{
			Type:      media.MesgVideo,
			Length:    naluHdrSize + naluLen,
			Timestamp: mesg.Timestamp,
			Data:      sharedBuf,
		}
		if naluType == naluTypeIDR {
			mesg.Flags |= media.FlagKeyFrame
		}
		if !firstMesg {
			mesg.Flags |= media.FlagContinuation
		}
		session.pub.SendMessage(&mesg)

		sharedBuf.Release()
		sharedBuf = nil
		firstMesg = false
	}

	return false, nil
}

func (t *Transport) onVideoMessage(mesg *rtmplib.VideoMessage, session *rtmpSession) error {
	hdr := flv.VideoTagHeader{}
	if err := hdr.Read(mesg.Data); err != nil {
		return fmt.Errorf("read video tag header: %w", err)
	}
	if hdr.Codec() != flv.VideoCodecH264 {
		return errInvalidVideoCodec
	}
	if hdr.PacketType() == flv.PacketSeqHeader {
		return t.sendSeqHeader(media.MesgVideoSeqHdr, mesg.Data, mesg.Length-uint32(flv.VideoTagHeaderSize), session)
	}
	if hdr.PacketType() == flv.PacketData {
		for {
			if done, err := t.sendVideoData(mesg, session); err != nil {
				return fmt.Errorf("send video data: %w", err)
			} else if done {
				return nil
			}
		}
	}
	return errInvalidFLVPacket
}

func (t *Transport) sendAudioData(mesg *rtmplib.AudioMessage, session *rtmpSession) error {
	var sharedBuf *media.SharedBuffer
	remaining := mesg.Length - uint32(flv.AudioTagHeaderSize)
	firstMesg := true

	for remaining > 0 {
		if sharedBuf == nil {
			sharedBuf = session.pub.AcquireBuffer()
		}

		n, err := sharedBuf.ReadN(mesg.Data, int(remaining))
		if err != nil {
			sharedBuf.Release()
			return fmt.Errorf("ReadN: %w", err)
		}
		remaining -= uint32(n)

		mesg := media.Message{
			Type:      media.MesgAudio,
			Timestamp: mesg.Timestamp,
			Length:    mesg.Length - uint32(flv.AudioTagHeaderSize),
			Data:      sharedBuf,
		}
		if !firstMesg {
			mesg.Flags |= media.FlagContinuation
		}
		session.pub.SendMessage(&mesg)

		sharedBuf.Release()
		sharedBuf = nil
		firstMesg = false
	}
	return nil
}

func (t *Transport) onAudioMessage(mesg *rtmplib.AudioMessage, session *rtmpSession) error {
	hdr := flv.AudioTagHeader{}
	if err := hdr.Read(mesg.Data); err != nil {
		return fmt.Errorf("read audio tag header: %w", err)
	}
	if hdr.Codec() != flv.AudioCodecAAC {
		return errInvalidAudioCodec
	}
	if hdr.PacketType() == flv.PacketSeqHeader {
		return t.sendSeqHeader(media.MesgAudioSeqHdr, mesg.Data, mesg.Length-uint32(flv.AudioTagHeaderSize), session)
	}
	if hdr.PacketType() == flv.PacketData {
		if err := t.sendAudioData(mesg, session); err != nil {
			return fmt.Errorf("send audio data: %w", err)
		}
		return nil
	}
	return errInvalidFLVPacket
}

func (t *Transport) onMetaDataMessage(mesg *rtmplib.MetaDataMessage, session *rtmpSession) error {
	meta := media.MetaData{
		VideoWidth:      mesg.Width,
		VideoHeight:     mesg.Height,
		VideoFrameRate:  mesg.FrameRate,
		VideoDataRate:   mesg.VideoDataRate,
		AudioChannels:   mesg.AudioChannels,
		AudioSampleRate: mesg.AudioSampleRate,
		AudioDataRate:   mesg.AudioDataRate,
	}
	return session.pub.SendMetaData(&meta)
}

func (t *Transport) onConnect(mesg *rtmplib.ConnectMessage, userData any) error {
	session, ok := userData.(*rtmpSession)
	if !ok {
		return errInvalidUserData
	}

	key := strings.TrimPrefix(mesg.AppName, "live/")
	var sk model.StreamKey
	// TODO: use hashed stream key
	if err := t.skService.GetByKey(session.ctx, key, &sk); err != nil {
		return fmt.Errorf("get stream key: %w", err)
	}

	id := media.PublisherID(strconv.FormatUint(sk.UserID, 10))
	session.pub = t.sender.NewPublisher(id)
	session.userID = sk.UserID

	return nil
}

func (t *Transport) onPublish(mesg *rtmplib.PublishStreamMessage, userData any) error {
	return nil
}

func (t *Transport) onConn(conn *rtmplib.Conn) {
	defer conn.Close()
	log.Printf("INFO: new RTMP conn: %s", conn.RemoteAddr().String())

	session := &rtmpSession{
		ctx:  context.TODO(),
		conn: conn,
	}
	stream, err := conn.AcceptStream(&rtmplib.AcceptStreamOptions{
		UserData:  session,
		OnConnect: t.onConnect,
		OnPublish: t.onPublish,
	})
	if err != nil {
		log.Printf("ERROR: accept stream: %v", err)
		return
	}

	if err := t.sender.AddPublisher(session.pub); err != nil {
		log.Printf("ERROR: add publisher: %v", err)
		return
	}
	defer t.sender.RemovePublisher(session.pub)
	log.Printf("INFO: added a publisher with ID: %s", session.pub.ID())

	if err := t.streamService.StartStream(session.ctx, session.userID); err != nil {
		log.Printf("ERROR: start RTMP stream: %v", err)
		return
	}

	for {
		mesg, err := conn.ReadStreamMessage(stream)
		if err != nil {
			if err == rtmplib.ErrUnsupportedMessage {
				continue
			}
			if err == rtmplib.ErrConnClosed {
				log.Printf("INFO: RTMP connection closed: %s", conn.RemoteAddr().String())
			} else {
				log.Printf("ERROR: read message: %v", err)
			}
			return
		}

		switch m := mesg.(type) {
		case *rtmplib.VideoMessage:
			err = t.onVideoMessage(m, session)
		case *rtmplib.AudioMessage:
			err = t.onAudioMessage(m, session)
		case *rtmplib.MetaDataMessage:
			err = t.onMetaDataMessage(m, session)
		case *rtmplib.CloseStreamMessage:
			log.Printf("INFO: stream %d with publisher ID %s closed", stream, session.pub.ID())
			return
		}

		if err != nil {
			log.Printf("ERROR: handle RTMP message: %v", err)
			return
		}
	}
}

func (t *Transport) RunServer(addr string, tlsConf *tls.Config) error {
	var ln rtmplib.Listener
	var err error
	if tlsConf == nil {
		ln, err = rtmplib.Listen(addr)
	} else {
		ln, err = rtmplib.ListenTLS(addr, tlsConf)
	}
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	t.ln = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("ERROR: accept RTMP conn: %v", err)
			continue
		}

		go t.onConn(conn)
	}

	return nil
}

func (t *Transport) StopServer() {
	t.ln.Close()
}
