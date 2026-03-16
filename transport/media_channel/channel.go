package media_channel

import (
	"sync"

	"lively/core/media"
)

type MediaChannel struct {
	publishers map[media.PublisherID]media.Publisher
	mu         sync.RWMutex
}

func New() *MediaChannel {
	return &MediaChannel{
		publishers: make(map[media.PublisherID]media.Publisher),
	}
}

func (mc *MediaChannel) NewPublisher(id media.PublisherID) media.Publisher {
	return newPublisher(id)
}

func (mc *MediaChannel) getPublisher(id media.PublisherID) media.Publisher {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.publishers[id]
}

func (mc *MediaChannel) AddPublisher(pub media.Publisher) error {
	if pub := mc.getPublisher(pub.ID()); pub != nil {
		return media.ErrPublisherExists
	}
	mc.mu.Lock()
	mc.publishers[pub.ID()] = pub
	mc.mu.Unlock()
	return nil
}

func (mc *MediaChannel) RemovePublisher(pub media.Publisher) {
	pub.Stop()
	mc.mu.Lock()
	delete(mc.publishers, pub.ID())
	mc.mu.Unlock()
}

func (mc *MediaChannel) AddConsumer(consumer media.Consumer) error {
	pub := mc.getPublisher(consumer.PublisherID())
	if pub == nil {
		return media.ErrNoPublisher
	}
	return pub.AddConsumer(consumer)
}

func (mc *MediaChannel) RemoveConsumer(consumer media.Consumer) {
	pub := mc.getPublisher(consumer.PublisherID())
	if pub != nil {
		pub.RemoveConsumer(consumer)
	}
}
