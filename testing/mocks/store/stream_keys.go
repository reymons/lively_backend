package mocks_store

import (
	"context"

	"lively/core/model"
	"lively/db"
	"lively/store"
)

type NewStreamKeysConfig struct {
	GetByKey    func(key string, sk *model.StreamKey) error
	GetByUserID func(userID uint64, sk *model.StreamKey) error
	Save        func(sk *model.StreamKey) error
}

type skStore struct {
	conf NewStreamKeysConfig
}

func NewStreamKeys(conf *NewStreamKeysConfig) store.StreamKeys {
	if conf == nil {
		conf = &NewStreamKeysConfig{}
	}
	return &skStore{conf: *conf}
}

func (s *skStore) GetByUserID(ctx context.Context, dbClient db.DB, userID uint64, sk *model.StreamKey) error {
	if s.conf.GetByUserID != nil {
		return s.conf.GetByUserID(userID, sk)
	}
	return errUnimplemented
}

func (s *skStore) GetByKey(ctx context.Context, dbClient db.DB, key string, sk *model.StreamKey) error {
	if s.conf.GetByKey != nil {
		return s.conf.GetByKey(key, sk)
	}
	return errUnimplemented
}

func (s *skStore) Save(ctx context.Context, dbClient db.DB, sk *model.StreamKey) error {
	if s.conf.Save != nil {
		return s.conf.Save(sk)
	}
	return errUnimplemented
}
