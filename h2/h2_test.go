package h2

import (
	"testing"

	"github.com/yurivish/toolkit/assert"
)

func assertInBin(t *testing.T, value, lower, binWidth uint64) {
	t.Helper()
	assert.True(t, lower <= value)
	assert.True(t, value < lower+binWidth)
}

func TestEncodeDecodeZero(t *testing.T) {
	e := Encoding{A: 2, B: 3}
	encoded := e.Encode64(0)
	lower, binWidth := e.Decode64(encoded)

	assert.Equal(t, lower, uint64(0))
	assert.Equal(t, binWidth, uint64(4))
}

func TestEncodeDecodeSmallValue(t *testing.T) {
	e := Encoding{A: 2, B: 3}
	value := uint64(10)
	encoded := e.Encode64(value)
	lower, binWidth := e.Decode64(encoded)

	assertInBin(t, value, lower, binWidth)
}

func TestEncodeDecodeLargeValue(t *testing.T) {
	e := Encoding{A: 4, B: 4}
	value := uint64(100000)
	encoded := e.Encode64(value)
	lower, binWidth := e.Decode64(encoded)

	assertInBin(t, value, lower, binWidth)
}

func TestMultipleValues(t *testing.T) {
	e := Encoding{A: 2, B: 3}
	values := []uint64{0, 1, 10, 100, 1000}
	for _, value := range values {
		encoded := e.Encode64(value)
		lower, binWidth := e.Decode64(encoded)
		assertInBin(t, value, lower, binWidth)
	}
}
