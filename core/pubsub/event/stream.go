package event

import "fmt"

const TopicStream = "stream"

type StreamStarted struct {
	UserID uint64 `json:"user_id"`
}

func (ev *StreamStarted) Topic() string {
	return fmt.Sprintf("stream.%d.started", ev.UserID)
}

func NewStreamStarted(userID uint64) *StreamStarted {
	return &StreamStarted{
		UserID: userID,
	}
}

type StreamViewers struct {
	UserID  uint64 `json:"user_id"`
	Viewers int    `json:"viewers"`
}

func (ev *StreamViewers) Topic() string {
	return fmt.Sprintf("stream.%d.viewers", ev.UserID)
}

func NewStreamViewers(userID uint64, viewers int) *StreamViewers {
	return &StreamViewers{
		UserID:  userID,
		Viewers: viewers,
	}
}
