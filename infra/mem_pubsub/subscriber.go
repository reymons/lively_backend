package mem_pubsub

import (
	"sync"

	"lively/core/pubsub"
)

const subEventsMaxBuf = 16

type Subscriber struct {
	events chan pubsub.Event
	bus    *Bus
	topics map[string]struct{}
	mu     sync.Mutex
}

func (s *Subscriber) Subscribe(topic string) {
	s.bus.subscribe(topic, s)
	s.mu.Lock()
	s.topics[topic] = struct{}{}
	s.mu.Unlock()
}

func (s *Subscriber) Unsubscribe(topic string) {
	s.bus.unsubscribe(topic, s)
	s.mu.Lock()
	delete(s.topics, topic)
	s.mu.Unlock()
}

func (s *Subscriber) Read() (pubsub.Event, error) {
	ev, open := <-s.events
	if !open {
		return nil, pubsub.ErrSubscriberClosed
	}
	return ev, nil
}

func (s *Subscriber) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for topic := range s.topics {
		s.bus.unsubscribe(topic, s)
	}
	clear(s.topics)
	close(s.events)
}
