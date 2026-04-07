package media

type ConsumerID string

type Consumer interface {
	ID() ConsumerID

	// Returns the ID of a publisher whom the consumer is connected to
	PublisherID() PublisherID

	Messages() chan<- Message

	// Stops the consumer from receiveing messages
	// This method must be idempotent
	Stop()
}
