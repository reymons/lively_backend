package core

import (
	"fmt"
	"log"
	"sync"
)

type consumerData struct {
	consumer          Consumer
	sentVideoKeyFrame bool
}

type publisher struct {
	id          PublisherID
	consumers   map[ConsumerID]consumerData
	mu          sync.RWMutex
	videoSeqHdr []byte
	audioSeqHdr []byte
}

func NewPublisher(id PublisherID) Publisher {
	return &publisher{id: id}
}

func (pub *publisher) ID() PublisherID {
	return pub.id
}

func (pub *publisher) hasConsumer(id ConsumerID) bool {
	pub.mu.RLock()
	defer pub.mu.RUnlock()
	_, ok := pub.consumers[id]
	return ok
}

func (pub *publisher) sendSeqHeaders(consumer Consumer) error {
	pub.mu.RLock()
	videoHdr := pub.videoSeqHdr
	audioHdr := pub.audioSeqHdr
	pub.mu.RUnlock()

	{
		frame := MediaFrame{Type: MediaFrameVideoSeqHdr, Data: videoHdr}
		if err := consumer.SendFrame(&frame); err != nil {
			return fmt.Errorf("send video sequence header: %w", err)
		}
	}

	{
		frame := MediaFrame{Type: MediaFrameAudioSeqHdr, Data: audioHdr}
		if err := consumer.SendFrame(&frame); err != nil {
			return fmt.Errorf("send audio sequence header: %w", err)
		}
	}

	return nil
}

func (pub *publisher) AddConsumer(consumer Consumer) error {
	if pub.hasConsumer(consumer.ID()) {
		return ErrConsumerExists
	}
	if err := pub.sendSeqHeaders(consumer); err != nil {
		return fmt.Errorf("send sequence headers: %w", err)
	}

	pub.mu.Lock()
	defer pub.mu.Unlock()
	if pub.consumers == nil {
		pub.consumers = make(map[ConsumerID]consumerData, 1)
	}
	pub.consumers[consumer.ID()] = consumerData{consumer: consumer}
	return nil
}

func (pub *publisher) RemoveConsumer(consumer Consumer) {
	pub.mu.Lock()
	delete(pub.consumers, consumer.ID())
	if len(pub.consumers) == 0 {
		pub.consumers = nil
	}
	pub.mu.Unlock()
}

func (pub *publisher) Stop() {
	pub.mu.RLock()
	defer pub.mu.RUnlock()

	for _, data := range pub.consumers {
		data.consumer.Stop()
	}
}

func (pub *publisher) SendFrame(frame *MediaFrame) error {
	// TODO: make sure header is not too huge in size
	if frame.Type == MediaFrameVideoSeqHdr {
		pub.mu.Lock()
		pub.videoSeqHdr = make([]byte, len(frame.Data))
		copy(pub.videoSeqHdr, frame.Data)
		pub.mu.Unlock()
	} else if frame.Type == MediaFrameAudioSeqHdr {
		pub.mu.Lock()
		pub.audioSeqHdr = make([]byte, len(frame.Data))
		copy(pub.audioSeqHdr, frame.Data)
		pub.mu.Unlock()
	}

	pub.mu.RLock()
	defer pub.mu.RUnlock()

	for id, data := range pub.consumers {
		if frame.Type == MediaFrameVideo && !frame.IsKey && !data.sentVideoKeyFrame {
			continue
		}

		if err := data.consumer.SendFrame(frame); err != nil {
			log.Printf("ERROR: send frame to consumer (id %s): %v", id, err)
		} else {
			if frame.Type == MediaFrameVideo && frame.IsKey && !data.sentVideoKeyFrame {
				data.sentVideoKeyFrame = true
				pub.consumers[id] = data
			}
		}
	}

	return nil
}

type MediaChannel struct {
	publishers map[PublisherID]Publisher
	mu         sync.RWMutex
}

func NewMediaChannel() *MediaChannel {
	return &MediaChannel{
		publishers: make(map[PublisherID]Publisher),
	}
}

func (mc *MediaChannel) getPublisher(id PublisherID) Publisher {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.publishers[id]
}

// Sender part

func (mc *MediaChannel) AddPublisher(pub Publisher) error {
	if pub := mc.getPublisher(pub.ID()); pub != nil {
		return ErrPublisherExists
	}
	mc.mu.Lock()
	mc.publishers[pub.ID()] = pub
	mc.mu.Unlock()
	return nil
}

func (mc *MediaChannel) RemovePublisher(pub Publisher) {
	pub.Stop()
	mc.mu.Lock()
	delete(mc.publishers, pub.ID())
	mc.mu.Unlock()
}

// Receiver part

func (mc *MediaChannel) AddConsumer(consumer Consumer) error {
	pub := mc.getPublisher(consumer.PublisherID())
	if pub == nil {
		return ErrNoPublisher
	}
	return pub.AddConsumer(consumer)
}

func (mc *MediaChannel) RemoveConsumer(consumer Consumer) {
	pub := mc.getPublisher(consumer.PublisherID())
	if pub != nil {
		pub.RemoveConsumer(consumer)
	}
}
