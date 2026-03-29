package media

import (
	"encoding/binary"
	"errors"
)

var (
	ErrBufferTooShort = errors.New("buffer is too short")
)

const (
	FrameVideo uint8 = iota
	FrameVideoSeqHdr
	FrameAudio
	FrameAudioSeqHdr
)

type Frame struct {
	Type      uint8
	Timestamp uint32
	Data      []byte
	IsKey     bool
}

type MetaData struct {
	VideoWidth      uint16
	VideoHeight     uint16
	VideoFrameRate  uint8
	VideoDataRate   uint16
	AudioChannels   uint8
	AudioSampleRate uint32
	AudioDataRate   uint16
}

const MetaDataMaxEncodedSize = 14

func (meta *MetaData) Encode(buf []byte) (int, error) {
	if len(buf) < MetaDataMaxEncodedSize {
		return 0, ErrBufferTooShort
	}
	binary.BigEndian.PutUint16(buf, meta.VideoWidth)
	binary.BigEndian.PutUint16(buf[2:], meta.VideoHeight)
	buf[4] = meta.VideoFrameRate
	binary.BigEndian.PutUint16(buf[5:], meta.VideoDataRate)
	buf[7] = meta.AudioChannels
	binary.BigEndian.PutUint32(buf[8:], meta.AudioSampleRate)
	binary.BigEndian.PutUint16(buf[12:], meta.AudioDataRate)
	return MetaDataMaxEncodedSize, nil
}
