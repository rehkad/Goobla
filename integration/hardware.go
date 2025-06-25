//go:build integration

package integration

import "github.com/goobla/goobla/discover"

// availableVRAM returns the maximum total memory across all detected GPUs.
// If no GPU is detected or discovery fails, zero is returned.
func availableVRAM() uint64 {
	gpus := discover.GetGPUInfo()
	var max uint64
	for _, g := range gpus {
		if g.Library == "cpu" {
			continue
		}
		if g.TotalMemory > max {
			max = g.TotalMemory
		}
	}
	return max
}
