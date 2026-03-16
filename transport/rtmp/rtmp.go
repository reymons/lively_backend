package rtmp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	rtmplib "github.com/reymons/rtmp-go"

	"lively/codec/flv"
	"lively/core/media"
	"lively/core/model"
	"lively/core/service"
)

var (
	errInvalidUserData = errors.New("userData is not a session")
)

type rtmpSession struct {
	conn        *rtmplib.Conn
	userID      uint64
	pub         media.Publisher
	ctx         context.Context
	naluLenSize uint8
}

type Transport struct {
	ln            rtmplib.Listener
	sender        media.Sender
	skService     service.StreamKey
	streamService service.Stream
}

func NewTransport(sender media.Sender, skService service.StreamKey, streamService service.Stream) *Transport {
	return &Transport{
		sender:        sender,
		skService:     skService,
		streamService: streamService,
	}
}

func (t *Transport) isAllowedNALU(nalu *flv.H264NALUnit) bool {
	return nalu.Type == flv.H264NALUTypeIDR || nalu.Type == flv.H264NALUTypeNonIDR
}

func (t *Transport) isKeyFrameNALU(nalu *flv.H264NALUnit) bool {
	return nalu.Type == flv.H264NALUTypeIDR
}

func (t *Transport) onVideoMessage(mesg *rtmplib.VideoMessage, session *rtmpSession) error {
	var tag flv.H264VideoTag
	if err := tag.Decode(mesg.Data); err != nil {
		return fmt.Errorf("decode flv video tag: %w", err)
	}

	if tag.PacketType == flv.H264PackTypeSeqHdr {
		var hdr flv.H264VideoSeqHeader
		if err := hdr.Decode(tag.Data); err != nil {
			return fmt.Errorf("decode sequence header: %w", err)
		}
		session.naluLenSize = hdr.NALULenSize
		frame := media.Frame{
			Type: media.FrameVideoSeqHdr,
			Data: tag.Data,
		}
		if err := session.pub.SendFrame(&frame); err != nil {
			return fmt.Errorf("send video seq header: %w", err)
		}
		return nil
	}

	if tag.PacketType == flv.H264PackTypeNALU {
		var itr flv.H264NALUIterator
		if err := flv.InitH264NALUIterator(&itr, session.naluLenSize, tag.Data); err != nil {
			return fmt.Errorf("init video itr: %w", err)
		}

		var unit flv.H264NALUnit
		for {
			unitBuf, err := itr.Walk(&unit)
			if err != nil {
				return fmt.Errorf("walk over NAL units: %w", err)
			}
			if len(unitBuf) < 1 {
				break
			}

			if t.isAllowedNALU(&unit) {
				frame := media.Frame{
					Type:      media.FrameVideo,
					Timestamp: mesg.Timestamp,
					Data:      unitBuf,
					IsKey:     t.isKeyFrameNALU(&unit),
				}
				if err := session.pub.SendFrame(&frame); err != nil {
					if err == media.ErrNoPublisher {
						return fmt.Errorf("send video data: %w", err)
					}
					log.Printf("WARNING: send nalu unit: %s, %v", session.pub.ID(), err)
				}
			}
		}
	}

	return nil
}

func (t *Transport) onAudioMessage(mesg *rtmplib.AudioMessage, session *rtmpSession) error {
	var tag flv.AACAudioTag
	if err := tag.Decode(mesg.Data); err != nil {
		return fmt.Errorf("decode flv audio tag: %w", err)
	}

	if tag.PacketType == flv.AACPackTypeSeqHdr {
		frame := media.Frame{
			Type: media.FrameAudioSeqHdr,
			Data: tag.Data,
		}
		if err := session.pub.SendFrame(&frame); err != nil {
			return fmt.Errorf("send audio seq header: %w", err)
		}
		return nil
	}

	// TODO: enable audio data later when I figure out the proper way of handling it on the client
	//if tag.PacketType == flv.AACPackTypeFrame {
	//	frame := media.Frame{
	//		Type:      media.FrameAudio,
	//		Timestamp: mesg.Timestamp,
	//		Data:      tag.Data,
	//	}
	//	if err := session.pub.SendFrame(&frame); err != nil {
	//		return fmt.Errorf("send audio frame: %w", err)
	//	}
	//}

	return nil
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
		case *rtmplib.CloseStreamMessage:
			log.Printf("INFO: stream %d with publisher ID %s closed", stream, session.pub.ID())
			return
		case *rtmplib.MetaDataMessage:
			log.Printf("INFO: meta: %+v", m)
		}

		if err != nil {
			log.Printf("ERROR: handle RTMP message: %v", err)

			if errors.Is(err, media.ErrNoPublisher) {
				return
			}
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
