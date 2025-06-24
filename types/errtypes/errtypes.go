// Package errtypes contains custom error types
package errtypes

import (
	"fmt"
	"strings"
)

const (
	UnknownMooglaKeyErrMsg = "unknown ollama key"
	InvalidModelNameErrMsg = "invalid model name"
)

// TODO: This should have a structured response from the API
type UnknownMooglaKey struct {
	Key string
}

func (e *UnknownMooglaKey) Error() string {
	return fmt.Sprintf("unauthorized: %s %q", UnknownMooglaKeyErrMsg, strings.TrimSpace(e.Key))
}
