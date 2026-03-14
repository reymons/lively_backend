package ws_media

import (
	"encoding/binary"
	"errors"
)

var (
	ErrBufferTooShort = errors.New("buffer is too short")
)

const (
	PackVideoSeqHdr = iota
	PackVideoFrame
)

const packHdrSize = 5

type Packet struct {
	Type      uint8
	Timestamp uint32
	Data      []byte
}

func (pack *Packet) Encode(data []byte) (int, error) {
	size := len(pack.Data) + packHdrSize
	if len(data) < size {
		return 0, ErrBufferTooShort
	}
	data[0] = pack.Type
	binary.BigEndian.PutUint32(data[1:], pack.Timestamp)
	copy(data[5:], pack.Data)
	return size, nil
}
