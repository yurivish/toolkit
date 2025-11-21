package h2

import "math/bits"

// todo: make 32- and 64-bit versions. The a/b params can be u32 in both cases.
type Encoding struct {
	A, B uint64
}

func (e Encoding) Encode64(value uint64) uint64 {
	a, b := e.A, e.B
	c := a + b + 1
	if value < 1<<c {
		return value >> a
	}
	seg := uint64(bits.Len64(value))
	return value>>(seg-b+1) + (seg-c)<<b
}

func (e Encoding) Decode64(index uint64) (lower, binWidth uint64) {
	a, b := e.A, e.B
	c := a + b + 1
	binsBelowCutoff := uint64(1) << (c - a)
	if index < binsBelowCutoff {
		// we're in the linear section of the histogram: each bin is 2^a units wide
		lower = index << a
		binWidth = (index+1)<<a - 1
	} else {
		// we're in the log section of the histogram: 2^b bins per log segment
		seg := c + (index-binsBelowCutoff)>>b
		offset := index & ((1 << b) - 1)
		lower = 1<<seg + offset<<(seg-b)
		binWidth = 1<<(seg-b) - 1
	}
	return
}
