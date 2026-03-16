package mem_pubsub

import (
	"math/rand"
	"reflect"
	"sync"
	"testing"

	"lively/core/pubsub"
	"lively/core/pubsub/event"
)

func TestEvents_SubscribesToEvent(t *testing.T) {
	bus := NewBus()
	sub := bus.NewSubscriber()
	publishedEv := event.NewStreamStarted(rand.Uint64())
	callCount := 0

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			ev, err := sub.Read()
			if err != nil {
				if err == pubsub.ErrSubscriberClosed {
					return
				}
				t.Fatalf("read event: %v", err)
			}
			callCount += 1
			if !reflect.DeepEqual(ev, publishedEv) {
				t.Errorf("invalid event data: expected %+v, got %+v", publishedEv, ev)
			}
			return
		}
	}()

	sub.Subscribe(event.TopicStream)
	bus.Publish(publishedEv)
	wg.Wait()

	if callCount != 1 {
		t.Errorf("invalid call count: expected %d, got %d", 1, callCount)
	}
}
