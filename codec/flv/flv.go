package flv

import (
	"encoding/binary"
	"errors"
)

const (
	VideoCodecH264 uint8 = 7
)

const (
	H264PackTypeSeqHdr uint8 = iota
	H264PackTypeNALU
	H264PackTypeSeqEnd
)

var (
	ErrBufferNoSpace = errors.New("not enough space in the buffer")
	ErrInvalidCodec  = errors.New("invalid codec")
)

type VideoTag interface {
	FrameType() uint8

	Codec() uint8

	Decode(data []byte) (int, error)
}

type videoTag struct {
	frameType uint8
	codec     uint8
}

func (t *videoTag) FrameType() uint8 {
	return t.frameType
}

func (t *videoTag) Codec() uint8 {
	return t.codec
}

func (t *videoTag) Decode(data []byte) (int, error) {
	if len(data) < 1 {
		return 0, ErrBufferNoSpace
	}

	var off int

	t.frameType = (data[0] & 0b11110000) >> 4
	t.codec = data[0] & 0b00001111
	off += 1

	return off, nil
}

type H264VideoTag struct {
	vt videoTag

	PacketType      uint8
	CompositionTime uint32
}

func (t *H264VideoTag) Decode(data []byte) (int, error) {
	n, err := t.vt.Decode(data)
	if err != nil {
		return 0, err
	}
	if t.vt.Codec() != VideoCodecH264 {
		return 0, ErrInvalidCodec
	}
	off := n
	data = data[off:]
	if len(data) < 4 {
		return 0, ErrBufferNoSpace
	}
	t.PacketType = data[0]
	t.CompositionTime = decode3BytesBE(data[1:])
	off += 4
	return off, nil
}

func GetNALULenSizeFromSeqHdr(data []byte) (uint8, error) {
	if len(data) < 5 {
		return 0, ErrBufferNoSpace
	}
	return uint8(data[4]&0b00000011) + 1, nil
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
