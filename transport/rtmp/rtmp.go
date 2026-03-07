package rtmp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"strings"

	rtmplib "github.com/reymons/rtmp-go"

	"lively/codec/flv"
	"lively/core"
	"lively/core/model"
	"lively/db"
	"lively/store"
)

type rtmpSession struct {
	conn        *rtmplib.Conn
	userID      core.PublisherID
	ctx         context.Context
	naluLenSize uint8
}

type Transport struct {
	ln         rtmplib.Listener
	sender     core.MediaChannelSender
	dbClient   db.Client
	streamKeys store.StreamKeys
}

func NewTransport(sender core.MediaChannelSender, dbClient db.Client, streamKeys store.StreamKeys) *Transport {
	return &Transport{
		sender:     sender,
		dbClient:   dbClient,
		streamKeys: streamKeys,
	}
}

func (t *Transport) onVideoMessage(mesg *rtmplib.VideoMessage, session *rtmpSession) error {
	var tag flv.H264VideoTag
	n, err := tag.Decode(mesg.Data)
	if err != nil {
		return fmt.Errorf("decode flv tag: %w", err)
	}

	if tag.PacketType == flv.H264PackTypeSeqHdr {
		hdr := mesg.Data[n:]
		size, err := flv.GetNALULenSizeFromSeqHdr(hdr)
		if err != nil {
			return fmt.Errorf("get nalu len size: %w", err)
		}
		session.naluLenSize = size

		err = t.sender.SendVideoSeqHeader(session.userID, hdr)
		if err != nil {
			return fmt.Errorf("send video seq header: %w", err)
		}

		return nil
	}

	if tag.PacketType == flv.H264PackTypeNALU {
		var itr flv.H264NALUnitIterator
		if err := flv.InitH264NALUnitIterator(&itr, session.naluLenSize, mesg.Data[n:]); err != nil {
			return fmt.Errorf("init video itr: %w", err)
		}

		var unit flv.H264NALUnit
		for !itr.Walk(&unit) {
			isKeyFrame := unit.Type == flv.H264NALUTypeIDR
			if isKeyFrame || unit.Type == flv.H264NALUTypeNonIDR {
				err = t.sender.SendVideoData(session.userID, mesg.Timestamp, unit.Data, isKeyFrame)
				if err != nil {
					if err == core.ErrNoPublisher {
						return fmt.Errorf("send video data: %w", err)
					}
					log.Printf("WARNING: send nalu unit: %d, %w", session.userID, err)
				}
			}
		}
	}

	return nil
}

func (t *Transport) onConnect(mesg *rtmplib.ConnectMessage, userData any) error {
	session, ok := userData.(*rtmpSession)
	if !ok {
		return fmt.Errorf("userData is not a session")
	}

	key := strings.TrimPrefix(mesg.AppName, "live/")
	var sk model.StreamKey
	// TODO: use hashed stream key
	if err := t.streamKeys.GetByKey(session.ctx, t.dbClient, key, &sk); err != nil {
		return fmt.Errorf("get stream key: %w", err)
	}
	if !sk.Active {
		return fmt.Errorf("inactive stream key")
	}
	session.userID = core.PublisherID(sk.UserID)

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

	if err := t.sender.AddPublisher(session.userID); err != nil {
		log.Printf("ERROR: add publisher: %v", err)
		return
	}
	defer t.sender.RemovePublisher(session.userID)
	log.Printf("INFO: added a publisher with ID: %d", session.userID)

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
		case *rtmplib.CloseStreamMessage:
			log.Printf("INFO: stream %d with publisher ID %d closed", stream, session.userID)
			return
		}

		if err != nil {
			log.Printf("ERROR: handle RTMP message: %v", err)

			if errors.Is(err, core.ErrNoPublisher) {
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
