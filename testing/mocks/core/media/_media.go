package mocks_media

import (
	"errors"

	"lively/core"
)

var errUnimplemented = errors.New("unimplemented")

type NewChannelSenderConfig struct {
	SendVideoData      func(id core.PublisherID, timestamp uint32, data []byte, isKeyFrame bool) error
	SendVideoSeqHeader func(id core.PublisherID, data []byte) error
	AddPublisher       func(id core.PublisherID) error
	RemovePublisher    func(id core.PublisherID)
}

type sender struct {
	conf NewChannelSenderConfig
}

func NewChannelSender(conf *NewChannelSenderConfig) core.MediaChannelSender {
	if conf == nil {
		conf = &NewChannelSenderConfig{}
	}
	return &sender{*conf}
}

func (s *sender) SendVideoData(id core.PublisherID, timestamp uint32, data []byte, isKeyFrame bool) error {
	if s.conf.SendVideoData != nil {
		return s.conf.SendVideoData(id, timestamp, data, isKeyFrame)
	}
	return errUnimplemented
}

func (s *sender) SendVideoSeqHeader(id core.PublisherID, data []byte) error {
	if s.conf.SendVideoSeqHeader != nil {
		return s.conf.SendVideoSeqHeader(id, data)
	}
	return errUnimplemented
}

func (s *sender) AddPublisher(id core.PublisherID) error {
	if s.conf.AddPublisher != nil {
		return s.conf.AddPublisher(id)
	}
	return errUnimplemented
}

func (s *sender) RemovePublisher(id core.PublisherID) {
	if s.conf.RemovePublisher != nil {
		s.conf.RemovePublisher(id)
	}
}
