package core

import (
	"errors"
)

var (
	ErrNoPublisher     = errors.New("no publisher with the specified ID")
	ErrPublisherExists = errors.New("a publisher with the specified ID already exists")
	ErrConsumerExists  = errors.New("a consumer with the specified ID already exists")
)

const (
	MediaFrameVideo uint8 = iota
	MediaFrameAudio
)

type MediaFrame struct {
	Type      uint8
	Timestamp uint32
	Data      []byte
}

func (fr *MediaFrame) CopyTo(frame *MediaFrame) {
	frame.Type = fr.Type
	frame.Timestamp = fr.Timestamp
	frame.Data = make([]byte, len(fr.Data))
	copy(frame.Data, fr.Data)
}

type PublisherID uint64

type ConsumerID uint64

type Consumer interface {
	ID() ConsumerID

	SendFrame(frame *MediaFrame) error

	SendVideoSeqHeader(data []byte) error

	// Stops the consumer from receiving frames
	Stop()
}

type MediaChannelSender interface {
	// If the publisher by the specified ID doesn't exist, returns ErrNoPublisher
	SendVideoData(id PublisherID, timestamp uint32, data []byte, isKeyFrame bool) error

	// If the publisher by the specified ID doesn't exist, returns ErrNoPublisher
	SendVideoSeqHeader(id PublisherID, data []byte) error

	// If a publisher with the specified ID already exists, returns ErrPublisherExists
	AddPublisher(id PublisherID) error

	RemovePublisher(id PublisherID)
}

type MediaChannelReceiver interface {
	// If the publisher by the specified ID doesn't exist, returns ErrNoPublisher
	GetVideoSeqHeader(id PublisherID) ([]byte, error)

	// If the publisher by the specified ID doesn't exist, returns ErrNoPublisher
	GetLatestVideoKeyFrame(id PublisherID, frame *MediaFrame) error

	// Adds a consumer to the publisher
	// If a publisher by the specified ID doesn't exist, returns ErrNoPublisher
	// If a consumer by the specified ID already exists, returns ErrConsumerExists
	AddConsumer(id PublisherID, consumer Consumer) error

	// Removes a consumer from the publisher's consumers
	RemoveConsumer(pubID PublisherID, cnsID ConsumerID)
}
