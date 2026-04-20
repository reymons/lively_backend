package ws_media

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/gobwas/ws"

	"lively/core/media"
	"lively/core/service"
)

type wsHeader struct {
	len  int
	data [ws.MaxHeaderSize]byte
}

func (h *wsHeader) Write(data []byte) (int, error) {
	if len(data) > len(h.data) {
		return 0, io.ErrShortBuffer
	}
	copy(h.data[:], data)
	h.len = len(data)
	return h.len, nil
}

func (h *wsHeader) encode(mesg *media.Message) error {
	hdr := ws.Header{
		OpCode: ws.OpBinary,
		Fin:    true,
		Length: int64(mesg.Length + mesgHdrLen),
	}
	if err := ws.WriteHeader(h, hdr); err != nil {
		return fmt.Errorf("write ws header: %w", err)
	}
	return nil
}

func (h *wsHeader) bytes() []byte {
	return h.data[:h.len]
}

const mesgHdrLen = 6

type mesgHeader [mesgHdrLen]byte

func (h *mesgHeader) encode(mesg *media.Message) {
	h[0] = mesg.Type
	h[1] = mesg.Flags
	binary.BigEndian.PutUint32(h[2:], mesg.Timestamp)
}

type wsConsumer struct {
	id       media.ConsumerID
	pubID    media.PublisherID
	conn     net.Conn
	messages chan media.Message
	stopped  atomic.Uint32
	bufs     net.Buffers
	wsHdr    wsHeader
	mesgHdr  mesgHeader
}

func newWSConsumer(id media.ConsumerID, pubID media.PublisherID, conn net.Conn) *wsConsumer {
	return &wsConsumer{
		id:    id,
		pubID: pubID,
		conn:  conn,
		// For backpressure, let's use buffered channels for now
		messages: make(chan media.Message, 128),
		bufs:     make(net.Buffers, 0, 3),
	}
}

func (c *wsConsumer) writeMessage(mesg *media.Message) error {
	defer mesg.Data.Release()

	if mesg.IsContinuation() {
		c.bufs = append(c.bufs, mesg.Data.Bytes())
	} else {
		if err := c.wsHdr.encode(mesg); err != nil {
			return fmt.Errorf("encode ws header: %w", err)
		}
		c.mesgHdr.encode(mesg)
		c.bufs = append(c.bufs, c.wsHdr.bytes(), c.mesgHdr[:], mesg.Data.Bytes())
	}

	if _, err := c.bufs.WriteTo(c.conn); err != nil {
		return fmt.Errorf("net.Buffers.WriteTo: %w", err)
	}
	c.bufs = c.bufs[:0]
	return nil
}

func (c *wsConsumer) readMessages(ready chan struct{}) {
	close(ready)
	defer c.conn.Close()

	for mesg := range c.messages {
		if err := c.writeMessage(&mesg); err != nil {
			log.Printf("ERROR: write ws message: %v", err)
			break
		}
	}

	for mesg := range c.messages {
		mesg.Data.Release()
	}
}

func (c *wsConsumer) run(receiver media.Receiver, streamService service.Stream, req *http.Request, userID uint64) {
	ready := make(chan struct{})
	go c.readMessages(ready)
	<-ready

	if err := receiver.AddConsumer(c); err != nil {
		log.Printf("ERROR: register consumer: %v", err)
		return
	}
	defer receiver.RemoveConsumer(c)
	log.Printf("INFO: added a consumer with id %s and pub id %s", c.id, c.pubID)

	// Handle viewer info
	if err := streamService.AddViewer(req.Context(), userID, req.RemoteAddr); err != nil {
		log.Printf("ERROR: add viewer: %v", err)
		return
	}
	defer func(streamService service.Stream, req *http.Request, userID uint64) {
		if err := streamService.RemoveViewer(req.Context(), userID, req.RemoteAddr); err != nil {
			log.Printf("ERROR: remove viewer: %v", err)
		}
	}(streamService, req, userID)

	buf := make([]byte, 1)
	for {
		if _, err := c.conn.Read(buf); err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				log.Printf("INFO: connection closed: %s", req.RemoteAddr)
			} else {
				log.Printf("ERROR: read WS conn data: %v", err)
			}
			return
		}
	}
}

func (c *wsConsumer) ID() media.ConsumerID {
	return c.id
}

func (c *wsConsumer) PublisherID() media.PublisherID {
	return c.pubID
}

func (c *wsConsumer) Messages() chan<- media.Message {
	return c.messages
}

func (c *wsConsumer) Stop() {
	if c.stopped.CompareAndSwap(0, 1) {
		close(c.messages)
	}
}
