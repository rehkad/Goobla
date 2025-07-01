package progress

import (
	"testing"
	"time"
)

func TestSpinnerStartStop(t *testing.T) {
	s := NewSpinner("test")
	defer s.Stop()
	time.Sleep(150 * time.Millisecond)

	s.mu.Lock()
	val1 := s.value
	s.mu.Unlock()
	if val1 == 0 {
		t.Fatalf("spinner did not advance")
	}

	s.Stop()
	val2 := s.value
	time.Sleep(150 * time.Millisecond)
	s.mu.Lock()
	if s.value != val2 {
		t.Fatalf("spinner advanced after stop")
	}
	s.mu.Unlock()
}
