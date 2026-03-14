package core

import (
	"errors"
)

var (
	ErrNoPublisher      = errors.New("no publisher with the specified ID")
	ErrPublisherExists  = errors.New("a publisher with the specified ID already exists")
	ErrConsumerExists   = errors.New("a consumer with the specified ID already exists")
	ErrUnsupportedFrame = errors.New("unsupported frame")
	ErrSendBufferFull   = errors.New("send buffer is full")
)

const (
	MediaFrameVideo uint8 = iota
	MediaFrameVideoSeqHdr
	MediaFrameAudio
	MediaFrameAudioSeqHdr
)

type MediaFrame struct {
	Type      uint8
	Timestamp uint32
	Data      []byte
	IsKey     bool
}

type PublisherID string

type ConsumerID string

type Publisher interface {
	ID() PublisherID

	AddConsumer(consumer Consumer) error

	RemoveConsumer(consumer Consumer)

	SendFrame(frame *MediaFrame) error

	Stop()
}

type Consumer interface {
	ID() ConsumerID

	PublisherID() PublisherID

	SendFrame(frame *MediaFrame) error

	Stop()
}

type MediaChannelSender interface {
	AddPublisher(pub Publisher) error

	RemovePublisher(pub Publisher)
}

type MediaChannelReceiver interface {
	AddConsumer(consumer Consumer) error

	RemoveConsumer(consumer Consumer)
}
