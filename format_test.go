package csid

import (
	"testing"
)

func TestFormat_validate(t *testing.T) {
	tests := []struct {
		name    string
		f       Format
		wantErr error
	}{
		{"default layout", Format{41, 12, 10}, nil},
		{"custom valid", Format{40, 13, 10}, nil},
		{"sum too low", Format{10, 10, 10}, ConfigErrInvalidBitFormat},
		{"sum too high", Format{41, 12, 11}, ConfigErrInvalidBitFormat},
		{"all zeros", Format{0, 0, 0}, ConfigErrInvalidBitFormat},
		{"single field", Format{63, 0, 0}, nil},
	}
	for _, tc := range tests {
		t.Run(
			tc.name, func(t *testing.T) {
				err := tc.f.validate()
				if err != tc.wantErr {
					t.Errorf("validate() = %v, want %v", err, tc.wantErr)
				}
			},
		)
	}
}

func TestFormat_compileFormat(t *testing.T) {
	c := DefaultFormat.compileFormat()

	// DefaultFormat: 41 timestamp, 12 node, 10 sequence
	// shiftNode = sequenceBits = 10
	// shiftTimestamp = sequenceBits + nodeBits = 22
	if c.shiftNode != 10 {
		t.Errorf("shiftNode = %d, want 10", c.shiftNode)
	}
	if c.shiftTimestamp != 22 {
		t.Errorf("shiftTimestamp = %d, want 22", c.shiftTimestamp)
	}
	if c.maxTimestamp != (1<<41)-1 {
		t.Errorf("maxTimestamp = %d, want %d", c.maxTimestamp, int64(1<<41)-1)
	}
	if c.maxNode != (1<<12)-1 {
		t.Errorf("maxNode = %d, want %d", c.maxNode, int64(1<<12)-1)
	}
	if c.maxSeq != (1<<10)-1 {
		t.Errorf("maxSeq = %d, want %d", c.maxSeq, int64(1<<10)-1)
	}
}

func TestFormat_compileFormat_Custom(t *testing.T) {
	f := Format{TimestampBits: 30, NodeBits: 23, SequenceBits: 10}
	c := f.compileFormat()

	if c.shiftNode != 10 {
		t.Errorf("shiftNode = %d, want 10", c.shiftNode)
	}
	if c.shiftTimestamp != 33 {
		t.Errorf("shiftTimestamp = %d, want 33", c.shiftTimestamp)
	}
	if c.maxTimestamp != (1<<30)-1 {
		t.Errorf("maxTimestamp = %d, want %d", c.maxTimestamp, int64(1<<30)-1)
	}
	if c.maxNode != (1<<23)-1 {
		t.Errorf("maxNode = %d, want %d", c.maxNode, int64(1<<23)-1)
	}
	if c.maxSeq != (1<<10)-1 {
		t.Errorf("maxSeq = %d, want %d", c.maxSeq, int64(1<<10)-1)
	}
}
