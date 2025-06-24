package sample

import (
	"fmt"
	"math/rand"
	"testing"
)

func BenchmarkWeightedSampler(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size %d", size), func(b *testing.B) {
			logits := make([]float32, size)
			for i := range logits {
				logits[i] = float32(rand.Float64()*10 - 5)
			}

			var sampler Sampler
			if err := json.Unmarshal([]byte(`{"temperature":0.8,"seed":42}`), &sampler); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()
			for b.Loop() {
				sampler.Sample(logits)
			}
		})
	}

	configs := []struct {
		name string
		json string
	}{
		{"Greedy", `{"temperature":0}`},
		{"Temperature", `{"temperature":0.8}`},
		{"TopK", `{"temperature":0.8,"top_k":50}`},
		{"TopP", `{"temperature":0.8,"top_p":0.9}`},
		{"MinP", `{"temperature":0.8,"min_p":0.05}`},
		{"WithSeed", `{"temperature":0.8,"top_k":50,"seed":42}`},
	}

	// Fixed size for common vocab size
	size := 128000
	logits := make([]float32, size)
	for i := range logits {
		logits[i] = float32(rand.Float64()*10 - 5)
	}

	for _, tc := range configs {
		b.Run("Config"+tc.name, func(b *testing.B) {
			var sampler Sampler
			if err := json.Unmarshal([]byte(tc.json), &sampler); err != nil {
				b.Fatal(err)
			}
			sampler.Sample(logits)

			b.ResetTimer()

			for b.Loop() {
				sampler.Sample(logits)
			}
		})
	}

	// Test with combined transforms separately - topK influences performance greatly
	b.Run("TransformCombined", func(b *testing.B) {
		var sampler Sampler
		if err := json.Unmarshal([]byte(`{"temperature":0.8,"top_k":50,"top_p":0.9,"min_p":0.05,"seed":42}`), &sampler); err != nil {
			b.Fatal(err)
		}
		b.ResetTimer()

		for b.Loop() {
			sampler.Sample(logits)
		}
	})
}

func BenchmarkGreedySampler(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000, 100000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size %d", size), func(b *testing.B) {
			logits := make([]float32, size)
			for i := range logits {
				logits[i] = float32(rand.Float64()*10 - 5)
			}

			var sampler Sampler
			if err := json.Unmarshal([]byte(`{"temperature":0}`), &sampler); err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()

			for b.Loop() {
				sampler.Sample(logits)
			}
		})
	}
}
