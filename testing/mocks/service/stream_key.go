package mocks_service

import (
	"context"

	"lively/core/model"
	"lively/core/service"
)

type StreamKey interface {
	GetByUserID(ctx context.Context, userID uint64, sk *model.StreamKey) error

	GetByKey(ctx context.Context, key string, sk *model.StreamKey) error
}

type NewStreamKeyConfig struct {
	GetByUserID func(userID uint64, sk *model.StreamKey) error
	GetByKey    func(key string, sk *model.StreamKey) error
}

type skService struct {
	conf NewStreamKeyConfig
}

func NewStreamKey(conf *NewStreamKeyConfig) service.StreamKey {
	if conf == nil {
		conf = &NewStreamKeyConfig{}
	}
	return &skService{conf: *conf}
}

func (s *skService) GetByUserID(ctx context.Context, userID uint64, sk *model.StreamKey) error {
	if s.conf.GetByUserID != nil {
		return s.conf.GetByUserID(userID, sk)
	}
	return errUnimplemented
}

func (s *skService) GetByKey(ctx context.Context, key string, sk *model.StreamKey) error {
	if s.conf.GetByKey != nil {
		return s.conf.GetByKey(key, sk)
	}
	return errUnimplemented
}
