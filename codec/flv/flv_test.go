package flv

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func getNALUnits(t *testing.T, lenSize uint8) ([]*H264NALUnit, []byte) {
	types := []uint8{
		H264NALUTypeIDR,
		H264NALUTypeNonIDR,
		H264NALUTypeSPS,
		H264NALUTypePPS,
	}
	units := make([]*H264NALUnit, 0, len(types))
	data := make([]byte, 4096)

	var off int
	for _, typ := range types {
		nalu := &H264NALUnit{
			Type: typ,
			Data: make([]byte, 64),
		}
		n, err := EncodeNALU(nalu, data[off:], lenSize)
		if err != nil {
			t.Fatalf("encode NALU: %v", err)
		}
		off += n
		units = append(units, nalu)
	}
	return units, data[:off]
}

func compareNALUnits(expected, got *H264NALUnit) error {
	if expected.Type != got.Type {
		return fmt.Errorf("invalid NALU type: expected %d, got %d", expected.Type, got.Type)
	}
	if !bytes.Equal(expected.Data, got.Data) {
		return fmt.Errorf("invalid NALU data: expected %x, got %x", expected.Data, got.Data)
	}
	return nil
}

func TestFLV_DecodeNALU(t *testing.T) {
	t.Parallel()

	type testCase struct {
		label       string
		nalu        *H264NALUnit
		encoded     []byte
		len         int
		lenSize     uint8
		expectedErr error
	}

	cases := []testCase{
		{
			label: "4-byte length",
			nalu: &H264NALUnit{
				Type: H264NALUTypeIDR,
				Data: []byte{1, 2, 3, 4},
			},
			encoded: []byte{0, 0, 0, 5, 5, 1, 2, 3, 4},
			len:     5,
			lenSize: 4,
		},
		{
			label: "2-byte length",
			nalu: &H264NALUnit{
				Type: H264NALUTypeNonIDR,
				Data: []byte{1, 2, 3, 4},
			},
			encoded: []byte{0, 5, 1, 1, 2, 3, 4},
			len:     5,
			lenSize: 2,
		},
		{
			label:       "5-byte length",
			nalu:        &H264NALUnit{},
			lenSize:     5,
			expectedErr: ErrInvalidNALULenSize,
		},
	}

	for _, tt := range cases {
		t.Run(tt.label, func(t *testing.T) {
			t.Parallel()

			nalu := &H264NALUnit{}
			naluLen, err := DecodeNALU(nalu, tt.encoded, tt.lenSize)
			if err != nil {
				if tt.expectedErr != nil {
					if !errors.Is(err, tt.expectedErr) {
						t.Errorf("invalid error: expected '%v', got '%v'", tt.expectedErr, err)
					}
					return
				}
				t.Fatalf("decode NALU: %v", err)
			}
			if naluLen != tt.len {
				t.Errorf("invalid NALU length: expected %d, got %d", tt.len, naluLen)
			}
			if nalu.Type != tt.nalu.Type {
				t.Errorf("invalid NALU type: expected %d, got %d", tt.nalu.Type, nalu.Type)
			}
			if !bytes.Equal(nalu.Data, tt.nalu.Data) {
				t.Errorf("invalid NALU data: expected %x, got %x", tt.nalu.Data, nalu.Data)
			}
		})
	}
}

func TestFLV_EncodeNALU(t *testing.T) {
	t.Parallel()

	type testCase struct {
		label       string
		nalu        *H264NALUnit
		encoded     []byte
		lenSize     uint8
		expectedErr error
	}

	cases := []testCase{
		{
			label: "4-byte length",
			nalu: &H264NALUnit{
				Type: H264NALUTypeIDR,
				Data: []byte{1, 2, 3, 4},
			},
			encoded: []byte{0, 0, 0, 5, 5, 1, 2, 3, 4},
			lenSize: 4,
		},
		{
			label: "2-byte length",
			nalu: &H264NALUnit{
				Type: H264NALUTypeNonIDR,
				Data: []byte{1, 2, 3, 4},
			},
			encoded: []byte{0, 5, 1, 1, 2, 3, 4},
			lenSize: 2,
		},
		{
			label: "5-byte length",
			nalu: &H264NALUnit{
				Type: H264NALUTypeNonIDR,
				Data: []byte{1, 2, 3, 4},
			},
			lenSize:     5,
			expectedErr: ErrInvalidNALULenSize,
		},
	}

	for _, tt := range cases {
		t.Run(tt.label, func(t *testing.T) {
			t.Parallel()

			data := make([]byte, 256)
			n, err := EncodeNALU(tt.nalu, data, tt.lenSize)
			if err != nil {
				if tt.expectedErr != nil {
					if !errors.Is(err, tt.expectedErr) {
						t.Errorf("invalid error: expected '%v', got '%v'", tt.expectedErr, err)
					}
					return
				}
				t.Fatalf("encode NALU: %v", err)
			}
			data = data[:n]
			if !bytes.Equal(tt.encoded, data) {
				t.Errorf("invalid encoded data: expected %x, got %x", tt.encoded, data)
			}
		})
	}
}

func TestFLV_H264NALUIterator_Walk(t *testing.T) {
	var itr H264NALUIterator
	lenSize := uint8(4)
	units, unitsBuf := getNALUnits(t, lenSize)
	if err := InitH264NALUIterator(&itr, lenSize, unitsBuf); err != nil {
		t.Fatalf("init iterator: %v", err)
	}
	var i int
	for {
		var got H264NALUnit
		unitBuf, err := itr.Walk(&got)
		if err != nil {
			t.Fatalf("walk: %v", err)
		}
		if len(unitBuf) < 1 {
			break
		}
		expected := units[i]
		i += 1
		if err := compareNALUnits(expected, &got); err != nil {
			t.Error(err)
		}
	}
}
