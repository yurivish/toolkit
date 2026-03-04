package exhaustigen

import (
	"fmt"
	"slices"
	"testing"
)

func TestElts(t *testing.T) {
	var g Gen
	count := 0
	for !g.Done() {
		elts := slices.Collect(GenElts(&g, 3, 4))
		fmt.Println(elts)
		count++
	}
	expected := 5*5*5 + 5*5 + 5 + 1
	if count != expected {
		t.Errorf("got %d, want %d", count, expected)
	}
}

func TestComb(t *testing.T) {
	var g Gen
	count := 0
	for !g.Done() {
		comb := slices.Collect(GenComb(&g, 5))
		fmt.Println(comb)
		count++
	}
	expected := 5*5*5*5*5 + 5*5*5*5 + 5*5*5 + 5*5 + 5 + 1
	if count != expected {
		t.Errorf("got %d, want %d", count, expected)
	}
}

func TestPerm(t *testing.T) {
	var g Gen
	count := 0
	for !g.Done() {
		perm := slices.Collect(GenPerm(&g, 5))
		fmt.Println(perm)
		count++
	}
	expected := 5 * 4 * 3 * 2 * 1
	if count != expected {
		t.Errorf("got %d, want %d", count, expected)
	}
}

func TestSubset(t *testing.T) {
	var g Gen
	count := 0
	for !g.Done() {
		subset := slices.Collect(GenSubset(&g, 5))
		fmt.Println(subset)
		count++
	}
	expected := 1 << 5
	if count != expected {
		t.Errorf("got %d, want %d", count, expected)
	}
}
