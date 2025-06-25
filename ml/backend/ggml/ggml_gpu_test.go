package ggml

import (
	"bytes"
	"os"
	"testing"

	"github.com/goobla/goobla/discover"
	ggmlfs "github.com/goobla/goobla/fs/ggml"
	"github.com/goobla/goobla/ml"
)

func TestVisionTensorGPUPlacement(t *testing.T) {
	if len(discover.GetGPUInfo().ByLibrary()) <= 1 {
		t.Skip("no GPU available")
	}

	f, err := os.CreateTemp(t.TempDir(), "model*.gguf")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	kv := ggmlfs.KV{
		"general.architecture": "test",
		"block_count":          uint32(0),
	}

	tensors := []*ggmlfs.Tensor{
		{Name: "v.patch_embd.weight", Shape: []uint64{1, 1}, WriterTo: bytes.NewBuffer(make([]byte, 1))},
	}

	if err := ggmlfs.WriteGGUF(f, kv, tensors); err != nil {
		t.Fatal(err)
	}

	b, err := New(f.Name(), ml.BackendParams{})
	if err != nil {
		t.Fatal(err)
	}

	mem := b.BackendMemory()
	if len(mem.GPUs) == 0 {
		t.Fatalf("no GPU memory entries")
	}
	if mem.GPUs[0].Weights[0].Size == 0 {
		t.Fatalf("vision tensor not allocated on GPU")
	}
	if mem.CPU.Weights[0].Size != 0 {
		t.Fatalf("vision tensor allocated on CPU")
	}
}
