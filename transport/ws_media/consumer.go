package ws_media

import (
	"log"

	ws "golang.org/x/net/websocket"

	"lively/core"
)

type wsConsumer struct {
	id     core.ConsumerID
	pubID  core.PublisherID
	conn   *ws.Conn
	sendCh chan []byte
}

func newWSConsumer(id core.ConsumerID, pubID core.PublisherID, conn *ws.Conn) *wsConsumer {
	return &wsConsumer{
		id:     id,
		pubID:  pubID,
		conn:   conn,
		sendCh: make(chan []byte),
	}
}

func (c *wsConsumer) readData() {
	for data := range c.sendCh {
		if err := ws.Message.Send(c.conn, data); err != nil {
			log.Printf("ERROR: send ws data: %v", err)
			continue
		}
	}
}

func (c *wsConsumer) ID() core.ConsumerID {
	return c.id
}

func (c *wsConsumer) PublisherID() core.PublisherID {
	return c.pubID
}

func (c *wsConsumer) sendPacket(pack *Packet) error {
	buf := [256 * 1024]byte{}
	n, err := pack.Encode(buf[:])
	if err != nil {
		return err
	}
	c.sendCh <- buf[:n]
	return nil
}

func (c *wsConsumer) SendFrame(frame *core.MediaFrame) error {
	pack := Packet{
		Timestamp: frame.Timestamp,
		Data:      frame.Data,
	}

	switch frame.Type {
	case core.MediaFrameVideo:
		pack.Type = PackVideoFrame
	case core.MediaFrameVideoSeqHdr:
		pack.Type = PackVideoSeqHdr
	default:
		return core.ErrUnsupportedFrame
	}

	return c.sendPacket(&pack)
}

func (c *wsConsumer) Stop() {
	c.conn.Close()
	close(c.sendCh)
}
