package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	rtmplib "github.com/reymons/rtmp-go"

	"lively/core"
	"lively/core/model"
	"lively/db"
	"lively/store"
)

type rtmpSession struct {
	userID core.PublisherID
	ctx    context.Context
}

type RTMP struct {
	ln         rtmplib.Listener
	sender     core.MediaChannelSender
	dbClient   db.Client
	streamKeys store.StreamKeys
}

func NewRTMP(sender core.MediaChannelSender, dbClient db.Client, streamKeys store.StreamKeys) *RTMP {
	return &RTMP{
		sender:     sender,
		dbClient:   dbClient,
		streamKeys: streamKeys,
	}
}

func (t *RTMP) onVideoMessage(mesg *rtmplib.VideoMessage, session *rtmpSession) error {
	return nil
}

func (t *RTMP) onConnect(mesg *rtmplib.ConnectMessage, userData any) error {
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

func (t *RTMP) onPublish(mesg *rtmplib.PublishStreamMessage, userData any) error {
	return nil
}

func (t *RTMP) onConn(conn *rtmplib.Conn) {
	defer conn.Close()
	log.Printf("INFO: new RTMP conn: %s", conn.RemoteAddr().String())

	session := &rtmpSession{ctx: context.TODO()}
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
		}
	}
}

func (t *RTMP) RunServer(addr string, tlsConf *tls.Config) error {
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

func (t *RTMP) StopServer() {
	t.ln.Close()
}
