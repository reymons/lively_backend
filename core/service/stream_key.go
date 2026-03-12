package service

import (
	"context"

	"lively/core"
	"lively/core/model"
	"lively/db"
	"lively/store"
)

type StreamKey interface {
	GetByUserID(ctx context.Context, userID uint64, sk *model.StreamKey) error

	GetByKey(ctx context.Context, key string, sk *model.StreamKey) error
}

type skService struct {
	dbClient   db.Client
	streamKeys store.StreamKeys
}

func NewStreamKey(dbClient db.Client, streamKeys store.StreamKeys) StreamKey {
	return &skService{dbClient, streamKeys}
}

func (s *skService) copyStreamKey(from *model.StreamKey, to *model.StreamKey) error {
	if !from.Active {
		return core.ErrInactiveStreamKey
	}
	*to = *from
	return nil
}

func (s *skService) GetByUserID(ctx context.Context, userID uint64, sk *model.StreamKey) error {
	var key model.StreamKey
	if err := s.streamKeys.GetByUserID(ctx, s.dbClient, userID, &key); err != nil {
		return err
	}
	return s.copyStreamKey(&key, sk)
}

func (s *skService) GetByKey(ctx context.Context, key string, sk *model.StreamKey) error {
	var skTmp model.StreamKey
	if err := s.streamKeys.GetByKey(ctx, s.dbClient, key, &skTmp); err != nil {
		return err
	}
	return s.copyStreamKey(&skTmp, sk)
}
