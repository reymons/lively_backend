package media

type PublisherID string

type Publisher interface {
	ID() PublisherID

	AddConsumer(consumer Consumer) error

	RemoveConsumer(consumer Consumer)

	SendFrame(frame *Frame) error

	Stop()
}
