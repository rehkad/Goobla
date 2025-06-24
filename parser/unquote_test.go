package parser

import "testing"

func TestUnquote(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"triple", `"""hello"""`, "hello", true},
		{"double", `"hello"`, "hello", true},
		{"single", `'hello'`, "hello", true},
		{"noquote", `hello`, "hello", true},
		{"bad triple", `"""hello""`, "", false},
		{"bad double", `"hello`, "", false},
		{"bad single", `'hello`, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := unquote(tt.in)
			if ok != tt.ok {
				t.Fatalf("expected ok=%v got %v", tt.ok, ok)
			}
			if got != tt.want {
				t.Fatalf("expected %q got %q", tt.want, got)
			}
		})
	}
}
