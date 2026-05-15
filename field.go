package ceroid

type FieldType int

const (
	FieldTimestamp FieldType = iota
	FieldNode
	FieldSequence

	FieldDataCenter
	FieldWorker
	FieldMeta
)

type Field struct {
	Type FieldType
	Bits uint8
	Name string
}
