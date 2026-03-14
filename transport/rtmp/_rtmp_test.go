// TODO: rewrite this shit according to new API
package rtmp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	rtmplib "github.com/reymons/rtmp-go"

	"lively/codec/flv"
	"lively/core"
	"lively/core/model"
	"lively/testing/mocks/core/media"
	"lively/testing/mocks/service"
)

func getPublisherID() (core.PublisherID, uint64) {
	userID := rand.Uint64()
	return core.PublisherID(strconv.FormatUint(userID, 10)), userID
}

func newTransport(skConf *mocks_service.NewStreamKeyConfig, senderConf *mocks_media.NewChannelSenderConfig) *Transport {
	skService := mocks_service.NewStreamKey(skConf)
	sender := mocks_media.NewChannelSender(senderConf)
	return NewTransport(sender, skService)
}

func newSession() *rtmpSession {
	return &rtmpSession{ctx: context.TODO(), naluLenSize: 4}
}

func newConnectMesg() *rtmplib.ConnectMessage {
	return &rtmplib.ConnectMessage{}
}

func newVideoTag(t *testing.T, packType uint8) *flv.H264VideoTag {
	return &flv.H264VideoTag{
		VideoTagHeader: flv.VideoTagHeader{
			FrameType: flv.VideoFrameTypeKey,
			Codec:     flv.VideoCodecH264,
		},
		PacketType: packType,
	}
}

func withSeqHdr(t *testing.T, tag *flv.H264VideoTag) *flv.H264VideoTag {
	tag.Data = make([]byte, 128)
	hdr := flv.H264VideoSeqHeader{NALULenSize: 4}
	n, err := hdr.Encode(tag.Data)
	if err != nil {
		t.Fatalf("encode sequence header: %v", err)
	}
	tag.Data = tag.Data[:n]
	return tag
}

func getNALUnits(t *testing.T) ([]*flv.H264NALUnit, []byte) {
	types := []uint8{
		flv.H264NALUTypeIDR,
		flv.H264NALUTypeNonIDR,
		flv.H264NALUTypeSPS,
		flv.H264NALUTypePPS,
	}
	units := make([]*flv.H264NALUnit, 0, len(types))
	data := make([]byte, 4096)

	var off int
	for _, typ := range types {
		nalu := &flv.H264NALUnit{
			Type: typ,
			Data: make([]byte, 64),
		}
		n, err := flv.EncodeNALU(nalu, data[off:], 4)
		if err != nil {
			t.Fatalf("encode NALU: %v", err)
		}
		off += n
		units = append(units, nalu)
	}
	return units, data[:off]
}

func withNALUnits(t *testing.T, tag *flv.H264VideoTag) (*flv.H264VideoTag, []*flv.H264NALUnit) {
	units, data := getNALUnits(t)
	tag.Data = data
	return tag, units
}

func newVideoMesg(t *testing.T, tag *flv.H264VideoTag) *rtmplib.VideoMessage {
	mesg := &rtmplib.VideoMessage{
		Timestamp: rand.Uint32(),
		Data:      make([]byte, 128+len(tag.Data)),
	}
	if n, err := tag.Encode(mesg.Data); err != nil {
		t.Fatalf("encode video tag: %v", err)
	} else {
		mesg.Data = mesg.Data[:n]
	}
	return mesg
}

func isAllowedNALU(nalu *flv.H264NALUnit) bool {
	return nalu.Type == flv.H264NALUTypeIDR || nalu.Type == flv.H264NALUTypeNonIDR
}

func compareNALUnits(expected, got *flv.H264NALUnit) error {
	if expected.Type != got.Type {
		return fmt.Errorf("invalid NALU type: expected %d, got %d", expected.Type, got.Type)
	}
	if !bytes.Equal(expected.Data, got.Data) {
		return fmt.Errorf("invalid NALU data: expected %x, got %x", expected.Data, got.Data)
	}
	return nil
}

func TestRTMP_OnConnect_TypeAssertsUserData(t *testing.T) {
	transport := newTransport(nil, nil)
	if err := transport.onConnect(nil, 10); !errors.Is(err, errInvalidUserData) {
		t.Errorf("invalid error: expected %v, got %v", errInvalidUserData, err)
	}
}

func TestRTMP_OnConnect_SetsCorrectDataToSession(t *testing.T) {
	t.Parallel()

	pubID, userID := getPublisherID()
	transport := newTransport(&mocks_service.NewStreamKeyConfig{
		GetByKey: func(key string, sk *model.StreamKey) error {
			sk.UserID = userID
			sk.Active = true
			return nil
		},
	}, nil)
	session := newSession()
	if err := transport.onConnect(newConnectMesg(), session); err != nil {
		t.Fatalf("onConnect: %v", err)
	}
	if session.pubID != pubID {
		t.Errorf("invalid session publisher ID: expected %s, got %s", pubID, session.pubID)
	}
}

func TestRTMP_OnConnect_ChecksIfKeyActive(t *testing.T) {
	t.Parallel()

	transport := newTransport(&mocks_service.NewStreamKeyConfig{
		GetByKey: func(key string, sk *model.StreamKey) error {
			return core.ErrInactiveStreamKey
		},
	}, nil)

	expected := core.ErrInactiveStreamKey
	if err := transport.onConnect(newConnectMesg(), newSession()); !errors.Is(err, expected) {
		t.Fatalf("invalid error: expected %v, got %v", expected, err)
	}
}

func TestRTMP_OnConnect_PassesCorrectKeyToStore(t *testing.T) {
	t.Parallel()

	type testCase struct {
		appName     string
		expectedKey string
	}

	cases := []testCase{
		{"live/sk_123", "sk_123"},
		{"some_key", "some_key"},
		{"live/a/b/c", "a/b/c"},
		{"", ""},
	}

	for _, tt := range cases {
		t.Run(tt.appName, func(t *testing.T) {
			t.Parallel()

			session := newSession()
			connMesg := newConnectMesg()
			connMesg.AppName = tt.appName

			var passedKey string
			transport := newTransport(&mocks_service.NewStreamKeyConfig{
				GetByKey: func(key string, sk *model.StreamKey) error {
					sk.Active = true
					passedKey = key
					return nil
				},
			}, nil)

			if err := transport.onConnect(connMesg, session); err != nil {
				t.Fatalf("onConnect: %v", err)
			}

			if tt.expectedKey != passedKey {
				t.Errorf("invalid passed key: expected %s, got %s", tt.expectedKey, passedKey)
			}
		})
	}
}

func TestRTMP_OnVideoMesg_HandlesNoPublisherWhenSendingSeqHdr(t *testing.T) {
	transport := newTransport(nil, &mocks_media.NewChannelSenderConfig{
		SendVideoSeqHeader: func(id core.PublisherID, hdr []byte) error {
			return core.ErrNoPublisher
		},
	})

	tag := withSeqHdr(t, newVideoTag(t, flv.H264PackTypeSeqHdr))
	mesg := newVideoMesg(t, tag)
	if err := transport.onVideoMessage(mesg, newSession()); !errors.Is(err, core.ErrNoPublisher) {
		t.Errorf("invalid error: expected '%v', got '%v'", core.ErrNoPublisher, err)
	}
}

func TestRTMP_OnVideoMesg_SendsCorrectSeqHeader(t *testing.T) {
	tag := withSeqHdr(t, newVideoTag(t, flv.H264PackTypeSeqHdr))
	mesg := newVideoMesg(t, tag)
	session := newSession()
	session.naluLenSize = 0
	callCount := 0

	transport := newTransport(nil, &mocks_media.NewChannelSenderConfig{
		SendVideoSeqHeader: func(id core.PublisherID, hdr []byte) error {
			callCount += 1
			if id != session.pubID {
				t.Errorf("invalid publisher id: expected %s, got %s", session.pubID, id)
			}
			if !bytes.Equal(hdr, tag.Data) {
				t.Errorf("invalid sequence header: expected %x, got %x", tag.Data, hdr)
			}
			return nil
		},
	})

	if err := transport.onVideoMessage(mesg, session); err != nil {
		t.Fatalf("onVideoMessage: %v", err)
	}
	if callCount != 1 {
		t.Errorf("invalid call count: expected %d, got %d", 1, callCount)
	}
	if session.naluLenSize != 4 {
		t.Errorf("invalid nalu length size: expected %d, got %d", 4, session.naluLenSize)
	}
}

func TestRTMP_OnVideoMesg_HandlesNoPublisherWhenSendingNALUUnits(t *testing.T) {
	tag, _ := withNALUnits(t, newVideoTag(t, flv.H264PackTypeNALU))
	mesg := newVideoMesg(t, tag)
	session := newSession()

	transport := newTransport(nil, &mocks_media.NewChannelSenderConfig{
		SendVideoData: func(id core.PublisherID, timestamp uint32, data []byte, isKeyFrame bool) error {
			return core.ErrNoPublisher
		},
	})

	if err := transport.onVideoMessage(mesg, session); !errors.Is(err, core.ErrNoPublisher) {
		t.Errorf("invalid error: expected '%v', got '%v'", core.ErrNoPublisher, err)
	}
}

func TestRTMP_OnVideoMesg_SendsCorrectNALUnits(t *testing.T) {
	tag, oldUnits := withNALUnits(t, newVideoTag(t, flv.H264PackTypeNALU))
	mesg := newVideoMesg(t, tag)
	session := newSession()
	callCount := 0

	units := make([]*flv.H264NALUnit, 0, len(oldUnits))
	for _, unit := range oldUnits {
		if isAllowedNALU(unit) {
			units = append(units, unit)
		}
	}
	expectedCallCount := len(units)

	transport := newTransport(nil, &mocks_media.NewChannelSenderConfig{
		SendVideoData: func(id core.PublisherID, timestamp uint32, data []byte, isKeyFrame bool) error {
			var got flv.H264NALUnit
			if _, err := flv.DecodeNALU(&got, data, session.naluLenSize); err != nil {
				t.Fatalf("decode NALU: %v", err)
			}
			if !isAllowedNALU(&got) {
				t.Errorf("disallowed NALU: type %d", got.Type)
			}
			expected := units[callCount]
			callCount += 1
			if err := compareNALUnits(expected, &got); err != nil {
				t.Error(err)
			}
			return nil
		},
	})

	if err := transport.onVideoMessage(mesg, session); err != nil {
		t.Fatalf("onVideoMessage: %v", err)
	}
	if callCount != expectedCallCount {
		t.Errorf("invalid call count: expected %d, got %d", expectedCallCount, callCount)
	}
}
