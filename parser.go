package crid

import "time"

// ParsedID holds the decoded fields of a Cero ID.
type ParsedID struct {
	Time       time.Time
	Entity     EntityType
	Datacenter uint8
	IsProd     bool
	WorkerID   uint8
	Sequence   uint8
}

// Parse decodes id using DefaultFormat.
func Parse(id int64) ParsedID {
	return parseWith(id, defaultCompiled)
}

// ParseWith decodes id using the given Format.
func ParseWith(id int64, f Format) (ParsedID, error) {
	if err := f.Validate(); err != nil {
		return ParsedID{}, err
	}
	return parseWith(id, compileFormat(f)), nil
}

func parseWith(id int64, c compiled) ParsedID {
	ms := (id >> c.shiftTimestamp) & c.maskTimestamp
	return ParsedID{
		Time:       Epoch.Add(time.Duration(ms) * time.Millisecond),
		Entity:     EntityType((id >> c.shiftEntity) & c.maskEntity),
		Datacenter: uint8((id >> c.shiftDatacenter) & c.maskDatacenter),
		IsProd:     c.maskEnv > 0 && ((id>>c.shiftEnv)&c.maskEnv) == 1,
		WorkerID:   uint8((id >> c.shiftWorker) & c.maskWorker),
		Sequence:   uint8(id & c.maskSequence),
	}
}
