package goobla

import (
	"testing"

	"github.com/goobla/goobla/server/internal/cache/blob"
)

func TestParseChunk(t *testing.T) {
	tests := []struct {
		in  string
		out blob.Chunk
		ok  bool
	}{
		{"0-1", blob.Chunk{Start: 0, End: 1}, true},
		{"5-5", blob.Chunk{Start: 5, End: 5}, true},
		{"1-0", blob.Chunk{}, false},
		{"a-b", blob.Chunk{}, false},
		{"1-b", blob.Chunk{}, false},
		{"1", blob.Chunk{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			c, err := parseChunk(tt.in)
			if (err == nil) != tt.ok {
				t.Fatalf("err=%v ok=%v", err, tt.ok)
			}
			if tt.ok && (c != tt.out) {
				t.Fatalf("got %+v want %+v", c, tt.out)
			}
		})
	}
}
