package ws_media

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	ws "golang.org/x/net/websocket"

	"lively/core"
)

var (
	errNoPublisherInfoInURL = errors.New("no publisher ID in URL")
)

type Transport struct {
	receiver core.MediaChannelReceiver
}

func NewTransport(receiver core.MediaChannelReceiver) *Transport {
	return &Transport{
		receiver: receiver,
	}
}

func (t *Transport) getPublisherID(url string) (core.PublisherID, error) {
	strs := strings.Split(url, "/")
	if len(strs) != 4 {
		return core.PublisherID(""), errNoPublisherInfoInURL
	}
	return core.PublisherID(strs[3]), nil
}

func (t *Transport) sendInitialVideoData(consumer *wsConsumer) error {
	hdr, err := t.receiver.GetVideoSeqHeader(consumer.pubID)
	if err != nil {
		return fmt.Errorf("get video seq header: %w", err)
	}
	if err := consumer.SendVideoSeqHeader(hdr); err != nil {
		return fmt.Errorf("send video seq header: %w", err)
	}

	var frame core.MediaFrame
	if err := t.receiver.GetLatestVideoKeyFrame(consumer.pubID, &frame); err != nil {
		return fmt.Errorf("get latest video key frame: %w", err)
	}
	if err := consumer.SendFrame(&frame); err != nil {
		return fmt.Errorf("send latest video key frame: %w", err)
	}

	return nil
}

func (t *Transport) onConn(conn *ws.Conn) {
	defer conn.Close()

	pubID, err := t.getPublisherID(conn.Request().URL.Path)
	if err != nil {
		log.Printf("ERROR: get publisher ID: %v", err)
		return
	}

	id, err := uuid.NewRandom()
	if err != nil {
		log.Printf("ERROR: generate UUID: %v", err)
		return
	}
	cnsID := core.ConsumerID(id.String())
	consumer := newWSConsumer(cnsID, pubID, conn)

	if err := t.sendInitialVideoData(consumer); err != nil {
		log.Printf("ERROR: send initial video data: %v", err)
		return
	}

	if err := t.receiver.AddConsumer(consumer); err != nil {
		log.Printf("ERROR: register consumer: %v", err)
		return
	}
	defer t.receiver.RemoveConsumer(consumer)

	go consumer.readPublisherData()

	for {
		// Dummy reader for now since a user is not supposed to send any data
		var buf []byte
		if err := ws.Message.Receive(consumer.conn, &buf); err != nil {
			log.Printf("INFO: connection closed: %s", conn.Request().RemoteAddr)
			return
		}
	}
}

func (t *Transport) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler := ws.Handler(t.onConn)
	handler.ServeHTTP(w, req)
}
