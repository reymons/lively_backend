package media

import (
	"encoding/binary"
	"io"
)

const (
	MesgVideo uint8 = iota
	MesgVideoSeqHdr
	MesgAudio
	MesgAudioSeqHdr
	MesgMetaData
)

const (
	FlagKeyFrame     uint8 = 1
	FlagContinuation       = 1 << 1
)

type Message struct {
	Type      uint8
	Flags     uint8
	Timestamp uint32
	Length    uint32
	Data      *SharedBuffer
}

func (m *Message) IsKeyFrame() bool {
	return m.Flags&FlagKeyFrame != 0
}

func (m *Message) IsContinuation() bool {
	return m.Flags&FlagContinuation != 0
}

type MetaData struct {
	VideoFrameRate  uint8
	VideoDataRate   uint16
	VideoWidth      uint16
	VideoHeight     uint16
	AudioChannels   uint8
	AudioDataRate   uint16
	AudioSampleRate uint32
}

const MetaDataEncodedSize = 14

func (meta *MetaData) Encode(buf []byte) error {
	if len(buf) < MetaDataEncodedSize {
		return io.ErrShortBuffer
	}
	binary.BigEndian.PutUint16(buf, meta.VideoWidth)
	binary.BigEndian.PutUint16(buf[2:], meta.VideoHeight)
	buf[4] = meta.VideoFrameRate
	binary.BigEndian.PutUint16(buf[5:], meta.VideoDataRate)
	buf[7] = meta.AudioChannels
	binary.BigEndian.PutUint32(buf[8:], meta.AudioSampleRate)
	binary.BigEndian.PutUint16(buf[12:], meta.AudioDataRate)
	return nil
}
