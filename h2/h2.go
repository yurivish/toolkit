package h2

import (
	"math/bits"
)

// todo: make 32- and 64-bit versions. The a/b params can be u32 in both cases.
type Encoding struct {
	A, B uint64
}

func (e Encoding) Encode64(value uint64) uint64 {
	a, b, c := e.A, e.B, e.A+e.B+1
	if value < 1<<c {
		return value >> a
	}
	logSegment := uint64(bits.Len64(value) - 1)
	return value>>(logSegment-b) + (logSegment-c+1)<<b
}

// The upper edge is exclusive: The represented range is [lower, lower + binWidth)
func (e Encoding) Decode64(index uint64) (lower, binWidth uint64) {
	a, b, c := e.A, e.B, e.A+e.B+1
	binsBelowCutoff := uint64(1) << (c - a)
	if index < binsBelowCutoff {
		// we're in the linear section of the histogram: each bin is 2^a units wide
		lower = index << a
		binWidth = 1 << a
	} else {
		// we're in the log section of the histogram: 2^b bins per log segment
		logSegment := c + (index-binsBelowCutoff)>>b
		offset := index & ((1 << b) - 1)
		lower = 1<<logSegment + offset<<(logSegment-b)
		binWidth = 1 << (logSegment - b)
	}
	return
}
