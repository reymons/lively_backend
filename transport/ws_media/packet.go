package ws_media

import (
	"encoding/binary"
	"errors"
)

var (
	ErrBufferTooShort = errors.New("buffer is too short")
)

const (
	PackVideoFrame uint8 = iota
	PackVideoSeqHdr
	PackAudioFrame
	PackAudioSeqHdr
)

const packHdrSize = 5

func boolToU8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

type Packet struct {
	Type       uint8
	Timestamp  uint32
	IsKeyFrame bool
	Data       []byte
}

func (pack *Packet) Encode(data []byte) (int, error) {
	size := len(pack.Data) + packHdrSize
	if len(data) < size {
		return 0, ErrBufferTooShort
	}
	data[0] = ((pack.Type & 0b01111111) << 1) | boolToU8(pack.IsKeyFrame)
	binary.BigEndian.PutUint32(data[1:], pack.Timestamp)
	copy(data[5:], pack.Data)
	return size, nil
}
