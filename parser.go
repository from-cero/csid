package ceroid

import "time"

// Parser decodes IDs without requiring a running Node.
// Use it in read-path services that only need to inspect IDs, not generate them.
type Parser struct {
	cfg Config
	c   compiled
}

// NewParser creates a Parser configured with the given options.
// Returns an error if the format is invalid.
func NewParser(opt ...Option) (*Parser, error) {
	cfg := applyOptions(opt)
	if err := cfg.Format.validate(); err != nil {
		return nil, err
	}
	return &Parser{
		cfg: cfg,
		c:   cfg.Format.compileFormat(),
	}, nil
}

// Parse decodes an ID into its timestamp, node, and sequence components.
func (p *Parser) Parse(id ID) ParsedID {
	return parseWith(id, p.cfg.Epoch, p.c)
}

func parseWith(id ID, epoch time.Time, c compiled) ParsedID {
	idI64 := int64(id)
	ts := (idI64 >> c.shiftTimestamp) & c.maxTimestamp
	node := (idI64 >> c.shiftNode) & c.maxNode
	seq := idI64 & c.maxSeq

	return ParsedID{
		Timestamp: epoch.Add(time.Duration(ts) * time.Millisecond),
		Node:      node,
		Sequence:  seq,
	}
}
