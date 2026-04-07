package flv

import "io"

const AudioTagHeaderSize = 2
const VideoTagHeaderSize = 5

const (
	VideoCodecH264 = 7

	AudioCodecAAC = 10
)

const (
	PacketSeqHeader = 0
	PacketData      = 1
)

type AudioTagHeader struct {
	buf [AudioTagHeaderSize]byte
}

func (t *AudioTagHeader) Read(r io.Reader) error {
	_, err := io.ReadFull(r, t.buf[:])
	return err
}

func (t *AudioTagHeader) Codec() uint8 {
	return (t.buf[0] & 0b11110000) >> 4
}

func (t *AudioTagHeader) PacketType() uint8 {
	return t.buf[1]
}

type VideoTagHeader struct {
	buf [VideoTagHeaderSize]byte
}

func (t *VideoTagHeader) Read(r io.Reader) error {
	_, err := io.ReadFull(r, t.buf[:])
	return err
}

func (t *VideoTagHeader) Codec() uint8 {
	return t.buf[0] & 0b00001111
}

func (t *VideoTagHeader) PacketType() uint8 {
	return t.buf[1]
}
