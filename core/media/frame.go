package media

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
