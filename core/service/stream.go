package service

import (
	"context"
	"sync"

	"lively/core/pubsub"
	"lively/core/pubsub/event"
)

type Stream interface {
	StartStream(ctx context.Context, userID uint64) error

	AddViewer(ctx context.Context, streamerUserID uint64, viewerIP string) error

	RemoveViewer(ctx context.Context, streamerUserID uint64, viewerIP string) error
}

type streamService struct {
	events  pubsub.Publisher
	viewers map[uint64]map[string]int
	mu      sync.Mutex
}

func NewStream(events pubsub.Publisher) Stream {
	return &streamService{
		events:  events,
		viewers: make(map[uint64]map[string]int),
	}
}

func (s *streamService) StartStream(ctx context.Context, userID uint64) error {
	s.events.Publish(event.NewStreamStarted(userID))
	return nil
}

func (s *streamService) publishViewerEvent(userID uint64, viewers int) {
	s.events.Publish(event.NewStreamViewers(userID, viewers))
}

func (s *streamService) AddViewer(ctx context.Context, streamerUserID uint64, viewerIP string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	viewers := s.viewers[streamerUserID]
	if viewers == nil {
		viewers = make(map[string]int, 1)
		s.viewers[streamerUserID] = viewers
	}
	count := viewers[viewerIP] + 1
	viewers[viewerIP] = count
	s.publishViewerEvent(streamerUserID, len(viewers))
	return nil
}

func (s *streamService) RemoveViewer(ctx context.Context, streamerUserID uint64, viewerIP string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	viewers := s.viewers[streamerUserID]
	if viewers == nil {
		return nil
	}
	count := viewers[viewerIP] - 1
	if count > 0 {
		viewers[viewerIP] = count
	} else {
		delete(viewers, viewerIP)
		if len(viewers) == 0 {
			delete(s.viewers, streamerUserID)
		}
		s.publishViewerEvent(streamerUserID, len(viewers))
	}
	return nil
}
