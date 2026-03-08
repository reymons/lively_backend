package flv

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrBufferTooShort = errors.New("buffer is too short")
	ErrInvalidCodec   = errors.New("invalid codec")
)

const (
	VideoFrameTypeKey uint8 = iota + 1
	VideoFrameTypeInter
	VideoFrameTypeDisposable
)

const (
	VideoCodecH264 uint8 = 7
)

type VideoTagHeader struct {
	FrameType uint8
	Codec     uint8
}

func (t *VideoTagHeader) Encode(data []byte) (int, error) {
	if len(data) < 1 {
		return 0, ErrBufferTooShort
	}
	data[0] = (t.FrameType << 4) | (t.Codec & 0b00001111)
	return 1, nil
}

func (t *VideoTagHeader) Decode(data []byte) (int, error) {
	if len(data) < 1 {
		return 0, ErrBufferTooShort
	}
	t.FrameType = (data[0] & 0b11110000) >> 4
	t.Codec = data[0] & 0b00001111
	return 1, nil
}

const (
	H264PackTypeSeqHdr uint8 = iota
	H264PackTypeNALU
	H264PackTypeSeqEnd
)

type H264VideoTag struct {
	VideoTagHeader

	PacketType      uint8
	CompositionTime uint32
	Data            []byte
}

func (t *H264VideoTag) Encode(data []byte) (int, error) {
	n, err := t.VideoTagHeader.Encode(data)
	if err != nil {
		return 0, fmt.Errorf("encode header: %w", err)
	}
	off := n
	data = data[off:]
	if len(data) < 4+len(t.Data) {
		return 0, ErrBufferTooShort
	}
	data[0] = t.PacketType
	encode3BytesBE(data[1:], t.CompositionTime)
	copy(data[4:], t.Data)
	off += 4 + len(t.Data)
	return off, nil
}

func (t *H264VideoTag) Decode(data []byte) error {
	n, err := t.VideoTagHeader.Decode(data)
	if err != nil {
		return fmt.Errorf("decode header: %w", err)
	}
	if t.VideoTagHeader.Codec != VideoCodecH264 {
		return ErrInvalidCodec
	}
	data = data[n:]
	if len(data) < 4 {
		return ErrBufferTooShort
	}
	t.PacketType = data[0]
	t.CompositionTime = decode3BytesBE(data[1:])
	t.Data = data[4:]
	return nil
}

type H264VideoSeqHeader struct {
	NALULenSize uint8
}

func (h *H264VideoSeqHeader) Encode(data []byte) (int, error) {
	if len(data) < 5 {
		return 0, ErrBufferTooShort
	}
	data[4] = (h.NALULenSize - 1) & 0b00000011
	return 5, nil
}

func (h *H264VideoSeqHeader) Decode(data []byte) error {
	if len(data) < 5 {
		return ErrBufferTooShort
	}
	h.NALULenSize = uint8(data[4]&0b00000011) + 1
	return nil
}

type H264NALUnitIterator struct {
	off         int
	naluLenSize uint8
	data        []byte
}

func InitH264NALUnitIterator(itr *H264NALUnitIterator, naluLenSize uint8, data []byte) error {
	if naluLenSize != 2 && naluLenSize != 4 {
		return ErrInvalidNALULenSize
	}

	itr.naluLenSize = naluLenSize
	itr.data = data
	return nil
}

func (itr *H264NALUnitIterator) Walk(nalu *H264NALUnit) bool {
	if itr.off >= len(itr.data) {
		return true
	}

	data := itr.data[itr.off:]
	off := 0

	var naluLen int
	switch itr.naluLenSize {
	case 2:
		if len(data) < 2 {
			return true
		}
		naluLen = int(binary.BigEndian.Uint16(data))
		off += 2
	case 4:
		if len(data) < 4 {
			return true
		}
		naluLen = int(binary.BigEndian.Uint32(data))
		off += 4
	default:
		return true
	}

	if err := nalu.Decode(data[off : off+naluLen]); err != nil {
		return true
	}
	off += naluLen
	itr.off += off
	return false
}
