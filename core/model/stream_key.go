package model

import "time"

type StreamKey struct {
	ID        uint64
	Key       string
	UserID    uint64
	Active    bool
	CreatedAt time.Time
}
