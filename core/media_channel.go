package core

import (
	"log"
	"sync"
)

type publisher struct {
	id            PublisherID
	consumers     map[ConsumerID]Consumer
	mu            sync.RWMutex
	videoKeyFrame MediaFrame
	videoSeqHdr   []byte
}

func (pub *publisher) getConsumer(id ConsumerID) Consumer {
	pub.mu.RLock()
	defer pub.mu.RUnlock()
	if pub.consumers == nil {
		return nil
	}
	return pub.consumers[id]
}

func (pub *publisher) addConsumer(consumer Consumer) {
	pub.mu.Lock()
	defer pub.mu.Unlock()
	if pub.consumers == nil {
		pub.consumers = make(map[ConsumerID]Consumer, 1)
	}
	pub.consumers[consumer.ID()] = consumer
}

func (pub *publisher) removeConsumer(id ConsumerID) {
	pub.mu.Lock()
	defer pub.mu.Unlock()
	delete(pub.consumers, id)
	if len(pub.consumers) == 0 {
		pub.consumers = nil
	}
}

func (pub *publisher) sendFrame(fr *MediaFrame) {
	// TODO: should I lock bofore the entire loop?
	pub.mu.RLock()
	defer pub.mu.RUnlock()

	for id, consumer := range pub.consumers {
		if err := consumer.SendFrame(fr); err != nil {
			log.Printf("ERROR: send frame to consumer (id %s): %v", id, err)
		}
	}
}

type MediaChannel struct {
	publishers map[PublisherID]*publisher
	mu         sync.RWMutex
}

func NewMediaChannel() *MediaChannel {
	return &MediaChannel{
		publishers: make(map[PublisherID]*publisher),
	}
}

func (mc *MediaChannel) getPublisher(id PublisherID) *publisher {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.publishers[id]
}

// Sender part

func (mc *MediaChannel) AddPublisher(id PublisherID) error {
	if pub := mc.getPublisher(id); pub != nil {
		return ErrPublisherExists
	}
	mc.mu.Lock()
	mc.publishers[id] = &publisher{id: id}
	mc.mu.Unlock()
	return nil
}

func (mc *MediaChannel) RemovePublisher(id PublisherID) {
	mc.mu.Lock()
	delete(mc.publishers, id)
	mc.mu.Unlock()
}

func (mc *MediaChannel) SendVideoData(id PublisherID, timestamp uint32, data []byte, isKeyFrame bool) error {
	pub := mc.getPublisher(id)
	if pub == nil {
		return ErrNoPublisher
	}
	frame := MediaFrame{
		Type:      MediaFrameVideo,
		Timestamp: timestamp,
		Data:      data,
	}
	if isKeyFrame {
		pub.mu.Lock()
		frame.CopyTo(&pub.videoKeyFrame)
		pub.mu.Unlock()
	}
	pub.sendFrame(&frame)
	return nil
}

func (mc *MediaChannel) SendVideoSeqHeader(id PublisherID, data []byte) error {
	pub := mc.getPublisher(id)
	if pub == nil {
		return ErrNoPublisher
	}

	pub.mu.Lock()
	pub.videoSeqHdr = make([]byte, len(data))
	copy(pub.videoSeqHdr, data)
	pub.mu.Unlock()

	// TODO: should I lock bofore the entire loop?
	pub.mu.RLock()
	defer pub.mu.RUnlock()

	for id, consumer := range pub.consumers {
		if err := consumer.SendVideoSeqHeader(data); err != nil {
			log.Printf("ERROR: send video seq hdr to consumer (id %s): %v", id, err)
		}
	}

	return nil
}

// Receiver part

func (mc *MediaChannel) AddConsumer(consumer Consumer) error {
	pub := mc.getPublisher(consumer.PublisherID())
	if pub == nil {
		return ErrNoPublisher
	}
	if cns := pub.getConsumer(consumer.ID()); cns != nil {
		return ErrConsumerExists
	}
	pub.addConsumer(consumer)
	return nil
}

func (mc *MediaChannel) RemoveConsumer(consumer Consumer) {
	pub := mc.getPublisher(consumer.PublisherID())
	if pub != nil {
		pub.removeConsumer(consumer.ID())
	}
}

func (mc *MediaChannel) GetVideoSeqHeader(id PublisherID) ([]byte, error) {
	pub := mc.getPublisher(id)
	if pub == nil {
		return []byte{}, ErrNoPublisher
	}
	pub.mu.RLock()
	defer pub.mu.RUnlock()
	return pub.videoSeqHdr, nil
}

func (mc *MediaChannel) GetLatestVideoKeyFrame(id PublisherID, frame *MediaFrame) error {
	pub := mc.getPublisher(id)
	if pub == nil {
		return ErrNoPublisher
	}
	pub.mu.RLock()
	defer pub.mu.RUnlock()
	*frame = pub.videoKeyFrame
	return nil
}
