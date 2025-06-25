// Package errtypes contains custom error types
package errtypes

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	UnknownGooblaKeyErrMsg = "unknown goobla key"
	InvalidModelNameErrMsg = "invalid model name"
)

// UnknownGooblaKey represents an invalid Goobla API key error.
//
// It marshals to JSON as:
//
//	{"error": "unknown goobla key", "key": "<key>"}
type UnknownGooblaKey struct {
	Key string `json:"key"`
}

// MarshalJSON implements json.Marshaler so that the error message is included
// alongside the key field in JSON responses.
func (e UnknownGooblaKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error string `json:"error"`
		Key   string `json:"key"`
	}{Error: UnknownGooblaKeyErrMsg, Key: e.Key})
}

func (e *UnknownGooblaKey) Error() string {
	return fmt.Sprintf("unauthorized: %s %q", UnknownGooblaKeyErrMsg, strings.TrimSpace(e.Key))
}
