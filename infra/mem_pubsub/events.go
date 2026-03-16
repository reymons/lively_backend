package mem_pubsub

import (
	"sync"

	"lively/core/pubsub"
)

type Bus struct {
	topicSubs map[string]map[*Subscriber]struct{}
	mu        sync.RWMutex
}

func NewBus() *Bus {
	return &Bus{
		topicSubs: make(map[string]map[*Subscriber]struct{}),
	}
}

func (b *Bus) forEachSection(topic string, cb func(section string)) {
	for i := 0; i < len(topic); i++ {
		if topic[i] == '.' {
			section := topic[:i]
			cb(section)
		}
	}
	cb(topic)
}

func (b *Bus) subscribe(topic string, sub *Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.topicSubs[topic]
	if subs == nil {
		subs = make(map[*Subscriber]struct{}, 1)
		b.topicSubs[topic] = subs
	}
	subs[sub] = struct{}{}
}

func (b *Bus) unsubscribe(topic string, sub *Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.topicSubs[topic]
	if subs != nil {
		delete(subs, sub)
	}
}

func (b *Bus) Publish(ev pubsub.Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	b.forEachSection(ev.Topic(), func(section string) {
		subs := b.topicSubs[section]
		for sub := range subs {
			select {
			case sub.events <- ev:
			default:
			}
		}
	})
}

func (b *Bus) NewSubscriber() *Subscriber {
	return &Subscriber{
		bus:    b,
		events: make(chan pubsub.Event, subEventsMaxBuf),
		topics: make(map[string]struct{}),
	}
}
