package media

import "errors"

var ErrUnsupportedFrame = errors.New("unsupported frame")

type ConsumerID string

type Consumer interface {
	ID() ConsumerID

	// Returns the ID of a publisher whom the consumer is connected to
	PublisherID() PublisherID

	// The implementation may wish to return ErrUnsupportedFrame is a frame's type is unsupported
	SendFrame(frame *Frame) error

	// Stops the consumer from receiveing frames
	// The implementation may wish stop its underlying network connection
	Stop()
}
