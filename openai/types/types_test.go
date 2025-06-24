package types

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/goobla/goobla/api"
)

var False = false

func TestFromCompleteRequestPromptTypes(t *testing.T) {
	testCases := []struct {
		name   string
		req    CompletionRequest
		expect api.GenerateRequest
	}{
		{
			name: "string slice",
			req:  CompletionRequest{Model: "test-model", Prompt: []string{"Hello", "World"}},
			expect: api.GenerateRequest{
				Model:  "test-model",
				Prompt: "HelloWorld",
				Options: map[string]any{
					"frequency_penalty": float32(0),
					"presence_penalty":  float32(0),
					"temperature":       1.0,
					"top_p":             1.0,
				},
				Stream: &False,
			},
		},
		{
			name: "token slice",
			req:  CompletionRequest{Model: "test-model", Prompt: []int{1, 2, 3}},
			expect: api.GenerateRequest{
				Model:   "test-model",
				Context: []int{1, 2, 3},
				Options: map[string]any{
					"frequency_penalty": float32(0),
					"presence_penalty":  float32(0),
					"temperature":       1.0,
					"top_p":             1.0,
				},
				Stream: &False,
			},
		},
		{
			name: "nested token slice",
			req:  CompletionRequest{Model: "test-model", Prompt: [][]int{{1, 2, 3}}},
			expect: api.GenerateRequest{
				Model:   "test-model",
				Context: []int{1, 2, 3},
				Options: map[string]any{
					"frequency_penalty": float32(0),
					"presence_penalty":  float32(0),
					"temperature":       1.0,
					"top_p":             1.0,
				},
				Stream: &False,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := FromCompleteRequest(tc.req)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.expect, got); diff != "" {
				t.Fatalf("requests did not match: %s", diff)
			}
		})
	}
}
