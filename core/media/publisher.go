package media

type PublisherID string

type Publisher interface {
	ID() PublisherID

	// Adds a consumer to send frames to
	// Returns ErrConsumerExists if such a consumer has been already added
	AddConsumer(consumer Consumer) error

	RemoveConsumer(consumer Consumer)

	// Sends a frame to the publisher's consumers
	SendFrame(frame *Frame) error

	// Stops the publisher from sending further frames
	// The publisher must stop all its consumers as well
	Stop()
}
