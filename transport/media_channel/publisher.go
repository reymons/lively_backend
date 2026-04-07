package media_channel

import (
	"fmt"
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
	pool        *sync.Pool
	videoSeqHdr *media.SharedBuffer
	audioSeqHdr *media.SharedBuffer
	meta        *media.SharedBuffer
}

func newPublisher(id media.PublisherID) *publisher {
	pool := &sync.Pool{}
	pool.New = func() any {
		buf := make([]byte, 8192)
		return media.NewSharedBuffer(pool, buf, 0)
	}

	return &publisher{
		id:   id,
		pool: pool,
	}
}

func (pub *publisher) ID() media.PublisherID {
	return pub.id
}

func (pub *publisher) AcquireBuffer() *media.SharedBuffer {
	buf := pub.pool.Get().(*media.SharedBuffer)
	buf.Acquire(1)
	return buf
}

func (pub *publisher) holdBuffer(buf *media.SharedBuffer, copy bool) *media.SharedBuffer {
	result := buf
	if copy {
		result = buf.CopySlice()
	}
	result.Acquire(1)
	return result
}

func (pub *publisher) sendMessage(consumer media.Consumer, mesg *media.Message) bool {
	mesg.Data.Acquire(1)

	select {
	case consumer.Messages() <- *mesg:
		return true
	default:
		mesg.Data.Release()
		return false
	}
}

func (pub *publisher) sendMessageStop(consumer media.Consumer, mesg *media.Message) bool {
	if pub.sendMessage(consumer, mesg) {
		return true
	}
	pub.mu.Lock()
	pub.stopConsumer(consumer)
	pub.mu.Unlock()
	return false
}

func (pub *publisher) sendInitialData(consumer media.Consumer) error {
	pub.mu.RLock()
	videoHdr := pub.videoSeqHdr
	audioHdr := pub.audioSeqHdr
	meta := pub.meta
	pub.mu.RUnlock()

	if meta != nil {
		mesg := media.Message{
			Type:   media.MesgMetaData,
			Length: uint32(meta.Len()),
			Data:   meta,
		}
		if !pub.sendMessageStop(consumer, &mesg) {
			return fmt.Errorf("failed to send meta data")
		}
	}

	if videoHdr != nil {
		mesg := media.Message{
			Type:   media.MesgVideoSeqHdr,
			Length: uint32(videoHdr.Len()),
			Data:   videoHdr,
		}
		if !pub.sendMessageStop(consumer, &mesg) {
			return fmt.Errorf("failed to send video seq header")
		}
	}

	if audioHdr != nil {
		mesg := media.Message{
			Type:   media.MesgAudioSeqHdr,
			Length: uint32(audioHdr.Len()),
			Data:   audioHdr,
		}
		if !pub.sendMessageStop(consumer, &mesg) {
			return fmt.Errorf("failed to send audio seq header")
		}
	}

	return nil
}

func (pub *publisher) SendMessage(mesg *media.Message) {
	if mesg.Type == media.MesgVideoSeqHdr {
		pub.mu.Lock()
		pub.videoSeqHdr = pub.holdBuffer(mesg.Data, true)
		pub.mu.Unlock()
	} else if mesg.Type == media.MesgAudioSeqHdr {
		pub.mu.Lock()
		pub.audioSeqHdr = pub.holdBuffer(mesg.Data, true)
		pub.mu.Unlock()
	}

	var removed []media.Consumer

	pub.mu.RLock()
	{
		for id, data := range pub.consumers {
			if mesg.Type == media.MesgVideo && !data.sentVideoKeyFrame && (!mesg.IsKeyFrame() || mesg.IsContinuation()) {
				continue
			}

			if pub.sendMessage(data.consumer, mesg) {
				if mesg.Type == media.MesgVideo && mesg.IsKeyFrame() && !data.sentVideoKeyFrame {
					data.sentVideoKeyFrame = true
					pub.consumers[id] = data
				}
			} else {
				removed = append(removed, data.consumer)
			}
		}
	}
	pub.mu.RUnlock()

	pub.mu.Lock()
	for _, consumer := range removed {
		pub.stopConsumer(consumer)
	}
	pub.mu.Unlock()
}

func (pub *publisher) SendMetaData(metaData *media.MetaData) error {
	data := make([]byte, media.MetaDataEncodedSize)
	if err := metaData.Encode(data); err != nil {
		return fmt.Errorf("encode meta data: %w", err)
	}

	buf := media.NewSharedBuffer(pub.pool, data, len(data))
	meta := pub.holdBuffer(buf, false)
	pub.mu.Lock()
	pub.meta = meta
	pub.mu.Unlock()

	mesg := media.Message{
		Type:   media.MesgMetaData,
		Length: uint32(meta.Len()),
		Data:   meta,
	}
	pub.SendMessage(&mesg)
	return nil
}

func (pub *publisher) AddConsumer(consumer media.Consumer) error {
	if pub.hasConsumer(consumer.ID()) {
		return media.ErrConsumerExists
	}
	if err := pub.sendInitialData(consumer); err != nil {
		return fmt.Errorf("send initial data: %w", err)
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
	pub.stopConsumer(consumer)
	pub.mu.Unlock()
}

func (pub *publisher) hasConsumer(id media.ConsumerID) bool {
	pub.mu.RLock()
	_, ok := pub.consumers[id]
	pub.mu.RUnlock()
	return ok
}

func (pub *publisher) stopConsumer(consumer media.Consumer) {
	delete(pub.consumers, consumer.ID())
	if len(pub.consumers) == 0 {
		pub.consumers = nil
	}
	consumer.Stop()
}

func (pub *publisher) Stop() {
	pub.mu.RLock()
	defer pub.mu.RUnlock()

	for _, data := range pub.consumers {
		data.consumer.Stop()
	}
}
