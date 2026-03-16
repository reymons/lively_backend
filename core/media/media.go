package media

import "errors"

var (
	ErrNoPublisher      = errors.New("no publisher with the specified ID")
	ErrPublisherExists  = errors.New("a publisher with the specified ID already exists")
	ErrConsumerExists   = errors.New("a consumer with the specified ID already exists")
	ErrUnsupportedFrame = errors.New("unsupported frame")
	ErrSendBufferFull   = errors.New("send buffer is full")
)

type Sender interface {
	NewPublisher(id PublisherID) Publisher

	AddPublisher(pub Publisher) error

	RemovePublisher(pub Publisher)
}

type Receiver interface {
	AddConsumer(consumer Consumer) error

	RemoveConsumer(consumer Consumer)
}
