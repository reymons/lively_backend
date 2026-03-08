package flv

import (
	"encoding/binary"
	"errors"
)

const (
	H264NALUTypeNonIDR = 1
	H264NALUTypeIDR    = 5
	H264NALUTypeSEI    = 6
	H264NALUTypeSPS    = 7
	H264NALUTypePPS    = 8
	H264NALUTypeFilter = 12
)

var (
	ErrInvalidNALULenSize = errors.New("invalid NALU length size")
)

type H264NALUnit struct {
	Type uint8
	Data []byte
}

func (u *H264NALUnit) Decode(data []byte) error {
	if len(data) < 2 {
		return ErrBufferTooShort
	}
	u.Type = data[0] & 0b00011111
	u.Data = data
	return nil
}

func (u *H264NALUnit) Encode(buf []byte) (int, error) {
	encodedSize := 4 + len(u.Data)
	if len(buf) < encodedSize {
		return 0, ErrBufferTooShort
	}
	binary.BigEndian.PutUint32(buf, uint32(len(u.Data)))
	copy(buf[4:], u.Data)
	return encodedSize, nil
}
