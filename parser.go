package ceroid

import "time"

type Parser struct {
	cfg Config
	c   compiled
}

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
