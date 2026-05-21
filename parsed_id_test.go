package csid

import (
	"strings"
	"testing"
	"time"
)

func TestParsedID_String(t *testing.T) {
	p := ParsedID{
		Timestamp: time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC),
		Node:      3,
		Sequence:  7,
	}
	s := p.String()

	for _, sub := range []string{"timestamp:", "node:", "sequence:"} {
		if !strings.Contains(s, sub) {
			t.Errorf("String() = %q, missing %q", s, sub)
		}
	}
	if !strings.Contains(s, "3") {
		t.Errorf("String() = %q, missing node value 3", s)
	}
	if !strings.Contains(s, "7") {
		t.Errorf("String() = %q, missing sequence value 7", s)
	}
}
