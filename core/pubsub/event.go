package pubsub

type Event interface {
	Topic() string
}
