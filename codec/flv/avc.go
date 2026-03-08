package flv

const h264NALUHdrSize = 1

const (
	H264NALUTypeNonIDR = 1
	H264NALUTypeIDR    = 5
	H264NALUTypeSEI    = 6
	H264NALUTypeSPS    = 7
	H264NALUTypePPS    = 8
	H264NALUTypeFilter = 12
)

type H264NALUnit struct {
	Type uint8
	Data []byte
}

func (u *H264NALUnit) Decode(data []byte) error {
	if len(data) < h264NALUHdrSize {
		return ErrBufferTooShort
	}
	u.Type = data[0] & 0b00011111
	u.Data = data[1:]
	return nil
}

func (u *H264NALUnit) Encode(buf []byte) (int, error) {
	size := h264NALUHdrSize + len(u.Data)
	if len(buf) < size {
		return 0, ErrBufferTooShort
	}
	buf[0] = u.Type & 0b00011111
	copy(buf[1:], u.Data)
	return size, nil
}
