package csid

import (
	"encoding/json"
	"testing"
)

func TestID_Int64(t *testing.T) {
	id := ID(42)
	if id.Int64() != 42 {
		t.Errorf("Int64() = %d, want 42", id.Int64())
	}
}

func TestID_String(t *testing.T) {
	tests := []struct {
		id   ID
		want string
	}{
		{ID(0), "0"},
		{ID(1), "1"},
		{ID(123456789), "123456789"},
		{ID(9223372036854775807), "9223372036854775807"},
	}
	for _, tc := range tests {
		if got := tc.id.String(); got != tc.want {
			t.Errorf("ID(%d).String() = %q, want %q", tc.id, got, tc.want)
		}
	}
}

func TestID_MarshalJSON(t *testing.T) {
	id := ID(123)
	b, err := id.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	want := `"123"`
	if string(b) != want {
		t.Errorf("MarshalJSON() = %s, want %s", b, want)
	}
}

func TestID_UnmarshalJSON_Valid(t *testing.T) {
	var id ID
	if err := id.UnmarshalJSON([]byte(`"456"`)); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if id != ID(456) {
		t.Errorf("UnmarshalJSON() = %d, want 456", id)
	}
}

func TestID_UnmarshalJSON_UnquotedInteger(t *testing.T) {
	var id ID
	if err := id.UnmarshalJSON([]byte(`123`)); err == nil {
		t.Error("UnmarshalJSON() expected error for unquoted integer, got nil")
	}
}

func TestID_UnmarshalJSON_NonInteger(t *testing.T) {
	var id ID
	if err := id.UnmarshalJSON([]byte(`"abc"`)); err == nil {
		t.Error("UnmarshalJSON() expected error for non-integer string, got nil")
	}
}

func TestID_JSONRoundTrip(t *testing.T) {
	original := ID(7654321098)
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var decoded ID
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded != original {
		t.Errorf("round-trip: got %d, want %d", decoded, original)
	}
}
