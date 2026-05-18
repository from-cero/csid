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
	id_i64 := int64(id)
	ts := (id_i64 >> c.shiftTimestamp) & c.maxTimestamp
	node := (id_i64 >> c.shiftNode) & c.maxNode
	seq := id_i64 & c.maxSeq

	return ParsedID{
		Timestamp: epoch.Add(time.Duration(ts) * time.Millisecond),
		Node:      node,
		Sequence:  seq,
	}
}
