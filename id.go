package ceroid

import "strconv"

type ID int64 // ID is a 63-bit Snowflake-style distributed identifier.

// Int64 returns the ID as a plain int64.
func (id ID) Int64() int64 {
	return int64(id)
}

// String returns the ID as a decimal string.
func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// MarshalJSON encodes the ID as a quoted decimal string to avoid precision loss in JavaScript,
// which cannot represent 63-bit integers exactly.
func (id ID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.String() + `"`), nil
}
