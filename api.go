package ceroid

type Generator interface {
	Generate() ID
}

type Parser interface {
	Parse(id ID) Parts
}
