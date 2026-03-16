package pubsub

import "errors"

var (
	ErrSubscriberClosed = errors.New("subscriber is closed")
)

type Publisher interface {
	Publish(ev Event)
}

type Subscriber interface {
	Subscribe(topic string)

	Unsubscribe(topic string)

	Close()

	Read() (Event, error)
}
