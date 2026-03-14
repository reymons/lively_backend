package flv

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

var (
	ErrBufferTooShort        = errors.New("buffer is too short")
	ErrInvalidCodec          = errors.New("invalid codec")
	ErrInvalidNALULenSize    = errors.New("invalid NALU length size: must be 2 or 4")
	ErrNALUTooLarge          = errors.New("NALU is too large")
	ErrNALULenPrefixTooShort = errors.New("actual NALU length is greater than the length header can hold")
)

type AudioTagHeader struct {
	SoundFormat uint8
	SoundRate   uint8
	SoundSize   uint8
	SoundType   uint8
}

func (t *AudioTagHeader) Decode(data []byte) (int, error) {
	if len(data) < 1 {
		return 0, ErrBufferTooShort
	}
	t.SoundFormat = (data[0] & 0b11110000) >> 4
	t.SoundRate = (data[0] & 0b00001100) >> 2
	t.SoundSize = (data[0] & 0b00000010) >> 1
	t.SoundType = (data[0] & 0b00000001)
	return 1, nil
}

const (
	AACPackTypeSeqHdr uint8 = iota
	AACPackTypeFrame
)

type AACAudioTag struct {
	AudioTagHeader

	PacketType uint8
	Data       []byte
}

func (t *AACAudioTag) Decode(data []byte) error {
	n, err := t.AudioTagHeader.Decode(data)
	if err != nil {
		return fmt.Errorf("decode header: %w", err)
	}
	data = data[n:]
	if len(data) < 1 {
		return ErrBufferTooShort
	}
	t.PacketType = data[0]
	t.Data = data[1:]
	return nil
}

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

// Decodes length-prefixed NALU
// Returns length of the NALU
func DecodeNALU(nalu *H264NALUnit, data []byte, lenSize uint8) (int, error) {
	var off int
	var naluLen uint32

	if lenSize == 4 {
		if len(data) < 4 {
			return 0, ErrBufferTooShort
		}
		naluLen = binary.BigEndian.Uint32(data)
		off += 4
	} else if lenSize == 2 {
		if len(data) < 2 {
			return 0, ErrBufferTooShort
		}
		naluLen = uint32(binary.BigEndian.Uint16(data))
		off += 2
	} else {
		return 0, ErrInvalidNALULenSize
	}

	if uint64(naluLen) > uint64(math.MaxInt) {
		return 0, ErrNALUTooLarge
	}
	length := int(naluLen)
	data = data[off:]
	if len(data) < length {
		return 0, ErrBufferTooShort
	}
	if err := nalu.Decode(data[:length]); err != nil {
		return 0, fmt.Errorf("nalu.Decode: %w", err)
	}
	return length, nil
}

// Encodes length-prefixed NALU
func EncodeNALU(nalu *H264NALUnit, data []byte, lenSize uint8) (int, error) {
	naluLen := uint64(h264NALUHdrSize + len(nalu.Data))
	var off int
	if lenSize == 2 {
		if len(data) < 2 {
			return 0, ErrBufferTooShort
		}
		if naluLen > uint64(math.MaxUint16) {
			return 0, ErrNALULenPrefixTooShort
		}
		binary.BigEndian.PutUint16(data, uint16(naluLen))
		off += 2
	} else if lenSize == 4 {
		if len(data) < 4 {
			return 0, ErrBufferTooShort
		}
		if naluLen > uint64(math.MaxUint32) {
			return 0, ErrNALULenPrefixTooShort
		}
		binary.BigEndian.PutUint32(data, uint32(naluLen))
		off += 4
	} else {
		return 0, ErrInvalidNALULenSize
	}
	if n, err := nalu.Encode(data[off:]); err != nil {
		return 0, fmt.Errorf("nalu.Encode: %w", err)
	} else {
		return off + n, nil
	}
}

type H264NALUIterator struct {
	off         int
	naluLenSize uint8
	data        []byte
}

func InitH264NALUIterator(itr *H264NALUIterator, naluLenSize uint8, data []byte) error {
	if naluLenSize != 2 && naluLenSize != 4 {
		return ErrInvalidNALULenSize
	}
	itr.naluLenSize = naluLenSize
	itr.data = data
	return nil
}

// Walks over NAL units
// Returns an empty slice if there're no more NAL units
func (itr *H264NALUIterator) Walk(nalu *H264NALUnit) ([]byte, error) {
	if itr.off >= len(itr.data) {
		return []byte{}, nil
	}
	data := itr.data[itr.off:]
	naluLen, err := DecodeNALU(nalu, data, itr.naluLenSize)
	if err != nil {
		return []byte{}, fmt.Errorf("decode NALU: %w", err)
	}
	totalLen := int(itr.naluLenSize) + naluLen
	itr.off += totalLen
	return data[:totalLen], nil
}
