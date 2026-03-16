package media

type ConsumerID string

type Consumer interface {
	ID() ConsumerID

	PublisherID() PublisherID

	SendFrame(frame *Frame) error

	Stop()
}
