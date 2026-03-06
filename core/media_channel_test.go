package core

import (
	"bytes"
	crypto_rand "crypto/rand"
	"math/rand/v2"
	"reflect"
	"testing"
)

func getSeqHeader(t *testing.T) []byte {
	hdr := make([]byte, 128)
	if _, err := crypto_rand.Read(hdr); err != nil {
		t.Fatalf("read random data for seq header: %v", err)
	}
	return hdr
}

func getPublisherID() PublisherID {
	return PublisherID(rand.Uint64())
}

func getConsumerID() ConsumerID {
	return ConsumerID(rand.Uint64())
}

func getFrame(t *testing.T, typ uint8) *MediaFrame {
	frame := &MediaFrame{
		Type:      typ,
		Timestamp: rand.Uint32(),
		Data:      make([]byte, 128),
	}
	if _, err := crypto_rand.Read(frame.Data); err != nil {
		t.Fatalf("read random data for frame: %v", err)
	}
	return frame
}

func newMediaChannelWithConsumer(t *testing.T) (*MediaChannel, *publisher, Consumer) {
	pubID := getPublisherID()
	consumer := NewConsumer(getConsumerID())
	channel := NewMediaChannel()
	if err := channel.AddPublisher(pubID); err != nil {
		t.Fatalf("add publisher: %v", err)
	}
	if err := channel.AddConsumer(pubID, consumer); err != nil {
		t.Fatalf("add consumer: %v", err)
	}
	pub := channel.getPublisher(pubID)
	return channel, pub, consumer
}

type MockConsumer struct {
	id               ConsumerID
	onVideoFrame     func(frame *MediaFrame)
	onVideoSeqHeader func(hdr []byte)
}

func NewConsumer(id ConsumerID) *MockConsumer {
	return &MockConsumer{id: id}
}

func (c *MockConsumer) ID() ConsumerID {
	return c.id
}

func (c *MockConsumer) OnVideoFrame(fn func(frame *MediaFrame)) {
	c.onVideoFrame = fn
}

func (c *MockConsumer) OnVideoSeqHeader(fn func(hdr []byte)) {
	c.onVideoSeqHeader = fn
}

func (c *MockConsumer) SendFrame(frame *MediaFrame) error {
	if c.onVideoFrame != nil {
		c.onVideoFrame(frame)
	}
	return nil
}

func (c *MockConsumer) SendVideoSeqHeader(hdr []byte) error {
	if c.onVideoSeqHeader != nil {
		c.onVideoSeqHeader(hdr)
	}
	return nil
}

func (c *MockConsumer) Stop() {}

func TestMediaChannel_AddPublisher(t *testing.T) {
	t.Parallel()

	pubID := getPublisherID()
	channel := NewMediaChannel()

	if pub := channel.getPublisher(pubID); pub != nil {
		t.Fatalf("publisher exists, though it was not added")
	}
	if err := channel.AddPublisher(pubID); err != nil {
		t.Fatalf("add publisher: %v", err)
	}

	if pub := channel.getPublisher(pubID); pub == nil {
		t.Errorf("expected a publisher with ID %d", pubID)
	} else if pub.id != pubID {
		t.Errorf("publisher IDs differ: expected %d, got %d", pubID, pub.id)
	}
}

func TestMediaChannel_AddExistingPublisher(t *testing.T) {
	t.Parallel()

	pubID := getPublisherID()
	channel := NewMediaChannel()

	if err := channel.AddPublisher(pubID); err != nil {
		t.Fatalf("add publisher: %v", err)
	}
	if err := channel.AddPublisher(pubID); err != ErrPublisherExists {
		t.Errorf("expected ErrPublisherExists error, got '%v'", err)
	}
}

func TestMediaChannel_RemovePublisher(t *testing.T) {
	t.Parallel()

	pubID := getPublisherID()
	channel := NewMediaChannel()

	if err := channel.AddPublisher(pubID); err != nil {
		t.Fatalf("add publisher: %v", err)
	}

	channel.RemovePublisher(pubID)
	if pub := channel.getPublisher(pubID); pub != nil {
		t.Errorf("publisher with ID %d was not removed", pubID)
	}
}

func TestMediaChannel_RemoveNonexistentPublisher(t *testing.T) {
	t.Parallel()

	channel := NewMediaChannel()
	channel.RemovePublisher(getPublisherID())
	channel.RemovePublisher(getPublisherID())
	channel.RemovePublisher(getPublisherID())
}

func TestMediaChannel_AddConsumer(t *testing.T) {
	t.Parallel()

	_, pub, consumer := newMediaChannelWithConsumer(t)
	if cns := pub.getConsumer(consumer.ID()); cns != consumer {
		t.Errorf("invalid consumer was added: expected with ID %d, got %d", consumer.ID(), cns.ID())
	}
}

func TestMediaChannel_AddExistingConsumer(t *testing.T) {
	t.Parallel()

	channel, pub, consumer := newMediaChannelWithConsumer(t)
	if err := channel.AddConsumer(pub.id, consumer); err != ErrConsumerExists {
		t.Errorf("expected ErrConsumerExists error, got '%v'", err)
	}
}

func TestMediaChannel_RemoveConsumer(t *testing.T) {
	t.Parallel()

	channel, pub, consumer := newMediaChannelWithConsumer(t)
	channel.RemoveConsumer(pub.id, consumer.ID())
	if cns := pub.getConsumer(consumer.ID()); cns != nil {
		t.Errorf("consumer with ID %d was not removed", consumer.ID())
	}
}

func TestMediaChannel_RemoveNonexistentConsumer(t *testing.T) {
	t.Parallel()

	channel, pub, consumer := newMediaChannelWithConsumer(t)
	channel.RemoveConsumer(pub.id, consumer.ID())
	channel.RemoveConsumer(pub.id, consumer.ID())
	channel.RemoveConsumer(pub.id, consumer.ID())
}

func TestMediaChannel_SendVideoData(t *testing.T) {
	// TODO: test on multiple consumers
	t.Parallel()

	var recvFrame *MediaFrame
	frame := getFrame(t, MediaFrameVideo)
	channel, pub, cns := newMediaChannelWithConsumer(t)
	consumer := cns.(*MockConsumer)
	callCount := 0

	consumer.OnVideoFrame(func(frame *MediaFrame) {
		recvFrame = frame
		callCount += 1
	})

	if err := channel.SendVideoData(pub.id, frame.Timestamp, frame.Data, false); err != nil {
		t.Fatalf("send video data: %v", err)
	}
	if callCount != 1 {
		t.Errorf("invalid call count: expected 1, got %d", callCount)
	}

	callCount = 0
	if !reflect.DeepEqual(frame, recvFrame) {
		t.Errorf("frame mismatch: expected %+v, got %+v", frame, recvFrame)
	}

	frame = getFrame(t, MediaFrameVideo)
	if err := channel.SendVideoData(pub.id, frame.Timestamp, frame.Data, true); err != nil {
		t.Fatalf("send video data for key frame: %v", err)
	}
	if callCount != 1 {
		t.Errorf("invalid call count: expected 1, got %d", callCount)
	}

	var keyFrame MediaFrame
	if err := channel.GetLatestVideoKeyFrame(pub.id, &keyFrame); err != nil {
		t.Fatalf("get latest video key frame: %v", err)
	}
	if !reflect.DeepEqual(frame, &keyFrame) {
		t.Errorf("key frame mismatch: expected %+v, got %+v", frame, &keyFrame)
	}
}

func TestMediaChannel_SendVideoDataToNonexistentPublisher(t *testing.T) {
	t.Parallel()

	channel := NewMediaChannel()
	frame := getFrame(t, MediaFrameVideo)
	pubID := getPublisherID()

	if err := channel.SendVideoData(pubID, frame.Timestamp, frame.Data, false); err != ErrNoPublisher {
		t.Errorf("expected ErrNoPublisher error, got '%v'", err)
	}
	if err := channel.SendVideoData(pubID, frame.Timestamp, frame.Data, true); err != ErrNoPublisher {
		t.Errorf("expected ErrNoPublisher error, got '%v'", err)
	}
}

func TestMediaChannel_SendVideoSeqHeader(t *testing.T) {
	t.Parallel()

	channel, pub, cns := newMediaChannelWithConsumer(t)
	consumer := cns.(*MockConsumer)
	hdr := getSeqHeader(t)
	callCount := 0

	var recvHdr []byte
	consumer.OnVideoSeqHeader(func(hdr []byte) {
		recvHdr = hdr
		callCount += 1
	})

	if err := channel.SendVideoSeqHeader(pub.id, hdr); err != nil {
		t.Fatalf("send seq header: %v", err)
	}

	if callCount != 1 {
		t.Fatalf("invalid call count: expected %d, got %d", 1, callCount)
	}

	if !bytes.Equal(hdr, recvHdr) {
		t.Errorf("header mismatch: expected %x, got %x", hdr, recvHdr)
	}

	storedHdr, err := channel.GetVideoSeqHeader(pub.id)
	if err != nil {
		t.Fatalf("get video seq header: %v", err)
	}

	if !bytes.Equal(hdr, storedHdr) {
		t.Errorf("header mismatch: expected %x, got %x", hdr, storedHdr)
	}
}
