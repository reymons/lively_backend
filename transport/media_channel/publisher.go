package media_channel

import (
	"fmt"
	"log"
	"sync"

	"lively/core/media"
)

type consumerData struct {
	consumer          media.Consumer
	sentVideoKeyFrame bool
}

type publisher struct {
	id          media.PublisherID
	consumers   map[media.ConsumerID]consumerData
	mu          sync.RWMutex
	videoSeqHdr []byte
	audioSeqHdr []byte
}

func newPublisher(id media.PublisherID) *publisher {
	return &publisher{id: id}
}

func (pub *publisher) ID() media.PublisherID {
	return pub.id
}

func (pub *publisher) hasConsumer(id media.ConsumerID) bool {
	pub.mu.RLock()
	defer pub.mu.RUnlock()
	_, ok := pub.consumers[id]
	return ok
}

func (pub *publisher) sendSeqHeaders(consumer media.Consumer) error {
	pub.mu.RLock()
	videoHdr := pub.videoSeqHdr
	audioHdr := pub.audioSeqHdr
	pub.mu.RUnlock()

	{
		if len(videoHdr) > 0 {
			frame := media.Frame{Type: media.FrameVideoSeqHdr, Data: videoHdr}
			if err := consumer.SendFrame(&frame); err != nil {
				return fmt.Errorf("send video sequence header: %w", err)
			}
		}
	}

	{
		if len(audioHdr) > 0 {
			frame := media.Frame{Type: media.FrameAudioSeqHdr, Data: audioHdr}
			if err := consumer.SendFrame(&frame); err != nil {
				return fmt.Errorf("send audio sequence header: %w", err)
			}
		}
	}

	return nil
}

func (pub *publisher) AddConsumer(consumer media.Consumer) error {
	if pub.hasConsumer(consumer.ID()) {
		return media.ErrConsumerExists
	}
	if err := pub.sendSeqHeaders(consumer); err != nil {
		return fmt.Errorf("send sequence headers: %w", err)
	}

	pub.mu.Lock()
	defer pub.mu.Unlock()
	if pub.consumers == nil {
		pub.consumers = make(map[media.ConsumerID]consumerData, 1)
	}
	pub.consumers[consumer.ID()] = consumerData{consumer: consumer}
	return nil
}

func (pub *publisher) RemoveConsumer(consumer media.Consumer) {
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

func (pub *publisher) SendFrame(frame *media.Frame) error {
	// TODO: make sure header is not too huge in size
	if frame.Type == media.FrameVideoSeqHdr {
		pub.mu.Lock()
		pub.videoSeqHdr = make([]byte, len(frame.Data))
		copy(pub.videoSeqHdr, frame.Data)
		pub.mu.Unlock()
	} else if frame.Type == media.FrameAudioSeqHdr {
		pub.mu.Lock()
		pub.audioSeqHdr = make([]byte, len(frame.Data))
		copy(pub.audioSeqHdr, frame.Data)
		pub.mu.Unlock()
	}

	pub.mu.RLock()
	defer pub.mu.RUnlock()

	for id, data := range pub.consumers {
		if frame.Type == media.FrameVideo && !frame.IsKey && !data.sentVideoKeyFrame {
			continue
		}

		if err := data.consumer.SendFrame(frame); err != nil {
			log.Printf("ERROR: send frame to consumer (id %s): %v", id, err)
		} else {
			if frame.Type == media.FrameVideo && frame.IsKey && !data.sentVideoKeyFrame {
				data.sentVideoKeyFrame = true
				pub.consumers[id] = data
			}
		}
	}

	return nil
}
