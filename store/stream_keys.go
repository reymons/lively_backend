package store

import (
	"context"

	"lively/core/model"
	"lively/db"
)

type StreamKeys interface {
	GetByKey(ctx context.Context, dbClient db.DB, key string, sk *model.StreamKey) error

	GetByUserID(ctx context.Context, dbClient db.DB, userID uint64, sk *model.StreamKey) error

	Save(ctx context.Context, dbClient db.DB, sk *model.StreamKey) error
}

type skStore struct{}

func NewStreamKeys() StreamKeys {
	return &skStore{}
}

func (s *skStore) GetByKey(ctx context.Context, dbClient db.DB, key string, sk *model.StreamKey) error {
	row := dbClient.QueryRowContext(
		ctx,
		"SELECT id, stream_key, user_id, active, created_at FROM stream_keys WHERE stream_key = $1",
		key,
	)
	return db.MapError(row.Scan(&sk.ID, &sk.Key, &sk.UserID, &sk.Active, &sk.CreatedAt))
}

func (s *skStore) GetByUserID(ctx context.Context, dbClient db.DB, userID uint64, sk *model.StreamKey) error {
	row := dbClient.QueryRowContext(
		ctx,
		"SELECT id, stream_key, active, created_at FROM stream_keys WHERE user_id = $1",
		userID,
	)
	return db.MapError(row.Scan(&sk.ID, &sk.Key, &sk.Active, &sk.CreatedAt))
}

func (s *skStore) Save(ctx context.Context, dbClient db.DB, sk *model.StreamKey) error {
	row := dbClient.QueryRowContext(
		ctx, "INSERT INTO stream_keys(stream_key, user_id, active) VALUES ($1,$2,$3) RETURNING id, created_at",
		sk.Key, sk.UserID, sk.Active,
	)
	return db.MapError(row.Scan(&sk.ID, &sk.CreatedAt))
}
