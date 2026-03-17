package media

import "errors"

var (
	ErrNoPublisher     = errors.New("no publisher with the specified ID")
	ErrPublisherExists = errors.New("a publisher with the specified ID already exists")
	ErrConsumerExists  = errors.New("a consumer with the specified ID already exists")
)

type Sender interface {
	NewPublisher(id PublisherID) Publisher

	// Adds a publisher whom consumers can receive frames from
	// Returns ErrPublisherExists if such a publisher has been already added
	AddPublisher(pub Publisher) error

	RemovePublisher(pub Publisher)
}

type Receiver interface {
	// Adds a consumer to receive frames from a publisher
	// Returns ErrNoPublisherExists if a publisher whom a consumer is connected to doesn't exist
	// Returns ErrConsumerExists if such a consumer has been already added
	AddConsumer(consumer Consumer) error

	RemoveConsumer(consumer Consumer)
}
