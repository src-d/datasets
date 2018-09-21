package model

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"

	"gopkg.in/src-d/go-kallax.v1/types"
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
func (h SHA1) Value() (driver.Value, error) {
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

// SHA1List is a slice of SHA1 hashes. Use this instead of just []SHA1
// because kallax will convert that to JSON and this implements its own
// methods to be stored as text[].
type SHA1List []SHA1

// Value returns a driver Value.
func (h SHA1List) Value() (driver.Value, error) {
	var ls = make([]string, len(h))
	for i, hash := range h {
		ls[i] = hash.String()
	}
	return types.Slice(ls).Value()
}

// Scan assigns a value from a database driver.
func (h *SHA1List) Scan(v interface{}) error {
	var l []string
	if err := types.Slice(&l).Scan(v); err != nil {
		return err
	}

	var res = make([]SHA1, len(l))
	for i, h := range l {
		res[i] = NewSHA1(h)
	}

	*h = res
	return nil
}
