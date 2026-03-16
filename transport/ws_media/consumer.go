package ws_media

import (
	"log"

	ws "golang.org/x/net/websocket"

	"lively/core/media"
)

type wsConsumer struct {
	id     media.ConsumerID
	pubID  media.PublisherID
	conn   *ws.Conn
	sendCh chan []byte
}

func newWSConsumer(id media.ConsumerID, pubID media.PublisherID, conn *ws.Conn) *wsConsumer {
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

func (c *wsConsumer) ID() media.ConsumerID {
	return c.id
}

func (c *wsConsumer) PublisherID() media.PublisherID {
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

func (c *wsConsumer) SendFrame(frame *media.Frame) error {
	pack := Packet{
		Timestamp:  frame.Timestamp,
		Data:       frame.Data,
		IsKeyFrame: frame.IsKey,
	}

	switch frame.Type {
	case media.FrameVideo:
		pack.Type = PackVideoFrame
	case media.FrameVideoSeqHdr:
		pack.Type = PackVideoSeqHdr
	case media.FrameAudio:
		pack.Type = PackAudioFrame
	case media.FrameAudioSeqHdr:
		pack.Type = PackAudioSeqHdr
	default:
		return media.ErrUnsupportedFrame
	}

	return c.sendPacket(&pack)
}

func (c *wsConsumer) Stop() {
	c.conn.Close()
	close(c.sendCh)
}
