package server

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goobla/goobla/api"
	"github.com/goobla/goobla/discover"
	"github.com/goobla/goobla/fs/ggml"
	"github.com/goobla/goobla/llm"
	"github.com/goobla/goobla/server/internal/testutil"
)

func TestGenerateWarnThinkingDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mock := mockRunner{
		CompletionResponse: llm.CompletionResponse{
			Done:               true,
			DoneReason:         llm.DoneReasonStop,
			PromptEvalCount:    1,
			PromptEvalDuration: 1,
			EvalCount:          1,
			EvalDuration:       1,
		},
	}

	s := Server{
		sched: &Scheduler{
			pendingReqCh:  make(chan *LlmRequest, 1),
			finishedReqCh: make(chan *LlmRequest, 1),
			expiredCh:     make(chan *runnerRef, 1),
			unloadedCh:    make(chan any, 1),
			loaded:        make(map[string]*runnerRef),
			newServerFn:   newMockServer(&mock),
			getGpuFn:      discover.GetGPUInfo,
			getCpuFn:      discover.GetCPUInfo,
			reschedDelay:  250 * time.Millisecond,
			loadFn: func(req *LlmRequest, _ *ggml.GGML, _ discover.GpuInfoList, _ int) {
				time.Sleep(time.Millisecond)
				req.successCh <- &runnerRef{llama: &mock}
			},
		},
	}

	go s.sched.Run(t.Context())

	_, digest := createBinFile(t, ggml.KV{
		"general.architecture":          "llama",
		"llama.block_count":             uint32(1),
		"llama.context_length":          uint32(8192),
		"llama.embedding_length":        uint32(4096),
		"llama.attention.head_count":    uint32(32),
		"llama.attention.head_count_kv": uint32(8),
		"tokenizer.ggml.tokens":         []string{""},
		"tokenizer.ggml.scores":         []float32{0},
		"tokenizer.ggml.token_type":     []int32{0},
	}, []*ggml.Tensor{{Name: "token_embd.weight", Shape: []uint64{1}, WriterTo: bytes.NewReader(make([]byte, 4))}})

	w := createRequest(t, s.CreateHandler, api.CreateRequest{
		Model:    "test",
		Files:    map[string]string{"file.gguf": digest},
		Template: `{{- if .Prompt }}{{ .Prompt }} {{ end }}{{- if .Response }}{{ .Response }}{{ end }}`,
		Stream:   &stream,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	logger, buf := testutil.SlogBuffer()
	slog.SetDefault(logger)

	think := false
	w = createRequest(t, s.GenerateHandler, api.GenerateRequest{
		Model:  "test",
		Prompt: "Hello",
		Think:  &think,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if !bytes.Contains(buf.Bytes(), []byte("does not support thinking output")) {
		t.Fatalf("expected warning about thinking support, got: %s", buf.String())
	}
}
