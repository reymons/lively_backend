package ws_media

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	ws "golang.org/x/net/websocket"

	"lively/core/media"
	"lively/core/service"
)

var (
	errNoPublisherInfoInURL = errors.New("no publisher ID in URL")
)

type Transport struct {
	receiver      media.Receiver
	streamService service.Stream
}

func NewTransport(receiver media.Receiver, streamService service.Stream) *Transport {
	return &Transport{
		receiver:      receiver,
		streamService: streamService,
	}
}

func (t *Transport) getPublisherID(url string) (media.PublisherID, uint64, error) {
	strs := strings.Split(url, "/")
	if len(strs) != 4 {
		return media.PublisherID(""), 0, errNoPublisherInfoInURL
	}
	id := strs[3]
	userID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return media.PublisherID(""), 0, fmt.Errorf("parse user ID: %w", err)
	}
	return media.PublisherID(id), userID, nil
}

func (t *Transport) getIP(addr string) (string, error) {
	ip, _, err := net.SplitHostPort(addr)
	return ip, err
}

func (t *Transport) onConn(conn *ws.Conn) {
	defer conn.Close()

	pubID, userID, err := t.getPublisherID(conn.Request().URL.Path)
	if err != nil {
		log.Printf("ERROR: get publisher ID: %v", err)
		return
	}

	id, err := uuid.NewRandom()
	if err != nil {
		log.Printf("ERROR: generate UUID: %v", err)
		return
	}
	cnsID := media.ConsumerID(id.String())
	consumer := newWSConsumer(cnsID, pubID, conn)

	go consumer.readData()

	if err := t.receiver.AddConsumer(consumer); err != nil {
		log.Printf("ERROR: register consumer: %v", err)
		return
	}
	defer t.receiver.RemoveConsumer(consumer)
	log.Printf("INFO: added a consumer with id %s and pub id %s", consumer.ID(), consumer.PublisherID())

	// Handle viewer info
	req := conn.Request()
	ip, err := t.getIP(req.RemoteAddr)
	if err != nil {
		log.Printf("ERROR: get IP: %v", err)
		return
	}
	if err = t.streamService.AddViewer(req.Context(), userID, ip); err != nil {
		log.Printf("ERROR: add viewer: %v", err)
		return
	}
	defer func() {
		if err := t.streamService.RemoveViewer(req.Context(), userID, ip); err != nil {
			log.Printf("ERROR: remove viewer: %v", err)
		}
	}()

	for {
		// Dummy reader for now since a user is not supposed to send any data
		var buf []byte
		if err := ws.Message.Receive(consumer.conn, &buf); err != nil {
			log.Printf("INFO: WS connection closed: %s", conn.Request().RemoteAddr)
			return
		}
	}
}

func (t *Transport) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler := ws.Handler(t.onConn)
	handler.ServeHTTP(w, req)
}
