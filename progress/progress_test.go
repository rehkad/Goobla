package progress

import (
	"bytes"
	"testing"
	"time"
)

func TestStopAndClear(t *testing.T) {
	buf := &bytes.Buffer{}
	p := NewProgress(buf)
	p.Add("test", NewSpinner("loading"))
	time.Sleep(10 * time.Millisecond)

	if !p.StopAndClear() {
		t.Fatalf("expected progress to stop")
	}

	// Ensure multiple calls are safe
	if p.StopAndClear() {
		t.Fatalf("progress stopped twice")
	}
}
