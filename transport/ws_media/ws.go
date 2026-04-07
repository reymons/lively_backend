package ws_media

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gobwas/ws"
	"github.com/google/uuid"

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

func (t *Transport) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(req, w)
	if err != nil {
		log.Printf("ERROR: upgrade HTTP: %v", err)
		return
	}
	defer conn.Close()

	pubID, userID, err := t.getPublisherID(req.URL.Path)
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
	consumer.run(t.receiver, t.streamService, req, userID)
}
