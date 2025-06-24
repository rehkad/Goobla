package discover

import (
	"strconv"
	"strings"
)

// PerformanceScore attempts to parse the Compute string and return a numeric
// value representing the relative performance of a GPU. Unknown values return 0.
func PerformanceScore(g GpuInfo) float64 {
	var b strings.Builder
	for _, r := range g.Compute {
		if (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return 0
	}
	v, err := strconv.ParseFloat(b.String(), 64)
	if err != nil {
		return 0
	}
	return v
}
