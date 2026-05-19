package ceroid

import (
	"strconv"
	"time"
)

// ParsedID holds the decoded components of an ID.
type ParsedID struct {
	Timestamp time.Time
	Node      int64
	Sequence  int64
}

// String returns a human-readable representation of the parsed ID components.
func (p ParsedID) String() string {
	return "{timestamp: " + p.Timestamp.String() +
		", node: " + strconv.FormatInt(p.Node, 10) +
		", sequence: " + strconv.FormatInt(p.Sequence, 10) + "}"
}
