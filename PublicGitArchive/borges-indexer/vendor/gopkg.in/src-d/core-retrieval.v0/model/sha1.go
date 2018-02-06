package model

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
)

// SHA1 is a SHA-1 hash.
type SHA1 [20]byte

func NewSHA1(s string) SHA1 {
	b, _ := hex.DecodeString(s)

	var h SHA1
	copy(h[:], b)

	return h
}

// String representation from this SHA1
func (h SHA1) String() string {
	return hex.EncodeToString(h[:])
}

// Value returns a driver Value.
func (h *SHA1) Value() (driver.Value, error) {
	return h.String(), nil
}

// Scan assigns a value from a database driver.
func (h *SHA1) Scan(v interface{}) error {
	switch t := v.(type) {
	case []byte:
		return h.Scan(string(t))
	case string:
		*h = NewSHA1(t)
		return nil
	default:
		return fmt.Errorf("unable to scan value of type %T into SHA1", v)
	}
}
