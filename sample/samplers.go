package sample

import (
	"encoding/json"
	"errors"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/moogla/moogla/llama"
	"github.com/moogla/moogla/model"
)

// token represents information about a single token during sampling
type token struct {
	id    int32   // The token's unique identifier
	value float32 // The raw logit or probability from the model
}

type Sampler struct {
	rng         *rand.Rand
	topK        int
	topP        float32
	minP        float32
	temperature float32
	grammar     *GrammarSampler
}

// Config specifies the sampling options used to build a Sampler.  It is
// intended to be populated via JSON and passed to [NewSampler].
type Config struct {
	Temperature float32 `json:"temperature,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	TopP        float32 `json:"top_p,omitempty"`
	MinP        float32 `json:"min_p,omitempty"`
	Seed        int     `json:"seed,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler so a Sampler can be constructed
// directly from its JSON configuration.
func (s *Sampler) UnmarshalJSON(b []byte) error {
	sampler, err := NewSampler(b, nil)
	if err != nil {
		return err
	}
	*s = sampler
	return nil
}

func (s *Sampler) Sample(logits []float32) (int32, error) {
	if len(logits) == 0 {
		return -1, errors.New("sample: no logits provided to sample")
	}

	tokens := make([]token, len(logits))
	for i := range logits {
		tokens[i].id = int32(i)
		tokens[i].value = logits[i]
	}

	t, err := s.sample(tokens)
	if err != nil {
		return -1, err
	}

	if s.grammar != nil {
		// optimization: first check if the max logit is accepted by the grammar
		// if the max logit is rejected, apply the grammar to all logits (slower)
		top := []token{t}
		s.grammar.Apply(top)
		if !math.IsInf(float64(top[0].value), -1) {
			s.grammar.Accept(top[0].id)
			return top[0].id, nil
		}

		// since .sample has side effects of modifying the tokens
		// we need to reset them before applying the grammar and
		// sampling again
		for i := range logits {
			tokens[i].id = int32(i)
			tokens[i].value = logits[i]
		}
		s.grammar.Apply(tokens)
		t, err = s.sample(tokens)
		if err != nil {
			return -1, err
		}
		s.grammar.Accept(t.id)
	}

	return t.id, nil
}

// greedy returns the highest probability token from the tokens
func greedy(tokens []token) token {
	max := tokens[0]
	for i := 1; i < len(tokens); i++ {
		if tokens[i].value > max.value {
			max = tokens[i]
		}
	}

	return max
}

// sample returns the highest probability token from the tokens
// given sampler parameters. It also has side effects of modifying the tokens
func (s *Sampler) sample(tokens []token) (token, error) {
	if s.temperature == 0 {
		return greedy(tokens), nil
	}

	// topK also sorts the tokens in descending order of logits
	tokens = topK(tokens, s.topK)

	// scale and normalize the tokens in place
	temperature(tokens, s.temperature)
	softmax(tokens)

	tokens = topP(tokens, s.topP)
	tokens = minP(tokens, s.minP)

	var r float32
	if s.rng != nil {
		r = s.rng.Float32()
	} else {
		r = rand.Float32()
	}

	// Calculate cumulative sum of probabilities
	var sum float32
	for i := range tokens {
		sum += tokens[i].value
		tokens[i].value = sum
	}
	r *= tokens[len(tokens)-1].value

	idx, _ := slices.BinarySearchFunc(tokens, r, func(token token, target float32) int {
		if token.value < target {
			return -1
		}
		return 1
	})

	if math.IsNaN(float64(sum)) {
		return token{}, errors.New("sample: logits sum to NaN, check model output")
	}
	return tokens[idx], nil
}

// NewSampler unmarshals the provided JSON configuration and returns a sampler
// configured with those options.
func NewSampler(b []byte, grammar *GrammarSampler) (Sampler, error) {
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Sampler{}, err
	}
	return NewSamplerFromConfig(cfg, grammar), nil
}

// NewSamplerFromConfig returns a sampler configured with the provided options.
// The configuration is typically populated via JSON and passed through [Config].
func NewSamplerFromConfig(cfg Config, grammar *GrammarSampler) Sampler {
	var rng *rand.Rand
	if cfg.Seed != -1 {
		// PCG requires two parameters: sequence and stream
		// Use original seed for sequence
		sequence := uint64(cfg.Seed)
		// Use golden ratio hash to generate statistically independent seeds
		rng = rand.New(rand.NewPCG(sequence, sequence^0x9E3779B9))
	}
	if cfg.Temperature < 0.0 {
		cfg.Temperature = 0.0
	}

	if cfg.TopP < 0.0 {
		cfg.TopP = 0.0
	}
	if cfg.TopP >= 1.0 {
		cfg.TopP = 1.0
	}

	if cfg.MinP < 0.0 {
		cfg.MinP = 0.0
	}
	if cfg.MinP >= 1.0 {
		cfg.MinP = 1.0
	}

	return Sampler{
		rng:         rng,
		topK:        cfg.TopK,
		topP:        cfg.TopP,
		minP:        cfg.MinP,
		temperature: cfg.Temperature,
		grammar:     grammar,
	}
}

type GrammarSampler struct {
	grammar *llama.Grammar
}

func NewGrammarSampler(model model.TextProcessor, grammarStr string) (*GrammarSampler, error) {
	vocabIds := make([]uint32, len(model.Vocabulary().Values))
	pieces := make([]string, len(model.Vocabulary().Values))
	for i := range model.Vocabulary().Values {
		pieces[i], _ = model.Decode([]int32{int32(i)})
		vocabIds[i] = uint32(i)
	}

	grammar := llama.NewGrammar(grammarStr, vocabIds, pieces, model.Vocabulary().EOS)
	if grammar == nil {
		return nil, errors.New("sample: failed to initialize grammar")
	}

	return &GrammarSampler{grammar: grammar}, nil
}

func (g *GrammarSampler) Apply(tokens []token) {
	tds := make([]llama.TokenData, len(tokens))
	for i, token := range tokens {
		tds[i].ID = token.id
		tds[i].Logit = token.value
	}
	g.grammar.Apply(tds)

	for i := range tokens {
		tokens[i].value = tds[i].Logit
	}
}

func (g *GrammarSampler) Accept(token int32) {
	g.grammar.Accept(token)
}

func (g *GrammarSampler) Free() {
	g.grammar.Free()
}
