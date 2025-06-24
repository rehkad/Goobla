// Package errtypes contains custom error types
package errtypes

import (
	"fmt"
	"strings"
)

const (
	UnknownGooblaKeyErrMsg = "unknown goobla key"
	InvalidModelNameErrMsg = "invalid model name"
)

// TODO: This should have a structured response from the API
type UnknownGooblaKey struct {
	Key string
}

func (e *UnknownGooblaKey) Error() string {
	return fmt.Sprintf("unauthorized: %s %q", UnknownGooblaKeyErrMsg, strings.TrimSpace(e.Key))
}
