// exhaustigen implements a simple utility tool for exhaustive testing described by Alex Kladov
// on his blog: https://matklad.github.io//2021/11/07/generate-all-the-things.html
// This implementation is ported from a Rust implementation of the idea by the
// creator of Rust, Graydon Hoare: https://github.com/graydon/exhaustigen-rs/
package exhaustigen

import "iter"

// Gen exhaustively enumerates all combinations of nondeterministic choices.
// It tracks variable-radix digit counters, incrementing like an odometer
// on each Done() call.
type Gen struct {
	started bool
	v       [][2]int // each entry is [current, bound]
	p       int
}

// Done returns false while there are unexplored combinations. On each call,
// it advances the odometer. Call this in the head of a for loop.
func (g *Gen) Done() bool {
	if !g.started {
		g.started = true
		return false
	}
	for i := len(g.v) - 1; i >= 0; i-- {
		if g.v[i][0] < g.v[i][1] {
			g.v[i][0]++
			g.v = g.v[:i+1]
			g.p = 0
			return false
		}
	}
	return true
}

// Gen returns a value (eventually every value) between 0 and bound inclusive.
func (g *Gen) Gen(bound int) int {
	if g.p == len(g.v) {
		g.v = append(g.v, [2]int{0, 0})
	}
	g.p++
	g.v[g.p-1][1] = bound
	return g.v[g.p-1][0]
}

// Flip returns false, then true.
func (g *Gen) Flip() bool {
	return g.Gen(1) == 1
}

// Pick returns an index (eventually every index) into a collection of size n.
func (g *Gen) Pick(n int) int {
	return g.Gen(n - 1)
}

// GenBoundBy generates a variable-length sequence (up to bound) of values
// produced by calling f(gen).
func GenBoundBy[T any](g *Gen, bound int, f func(*Gen) T) iter.Seq[T] {
	fixed := g.Gen(bound)
	return GenFixedBy(g, fixed, f)
}

// GenFixedBy generates a fixed-length sequence of values produced by calling f(gen).
func GenFixedBy[T any](g *Gen, fixed int, f func(*Gen) T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for range fixed {
			if !yield(f(g)) {
				return
			}
		}
	}
}

// GenElts generates variable-length sequences of bounded ints.
// Length <= lenBound, each element <= eltBound.
func GenElts(g *Gen, lenBound, eltBound int) iter.Seq[int] {
	return GenBoundBy(g, lenBound, func(g *Gen) int {
		return g.Gen(eltBound)
	})
}

// GenComb generates variable-size combinations with replacement from n elements.
// Yields indices in [0, n). Equivalent to GenBoundComb(g, n, n).
func GenComb(g *Gen, n int) iter.Seq[int] {
	return GenBoundComb(g, n, n)
}

// GenBoundComb generates variable-size (up to bound) combinations with
// replacement from n elements. Yields indices in [0, n).
func GenBoundComb(g *Gen, bound, n int) iter.Seq[int] {
	fixed := g.Gen(bound)
	return GenFixedComb(g, fixed, n)
}

// GenFixedComb generates fixed-size combinations with replacement from n elements.
// Yields indices in [0, n).
func GenFixedComb(g *Gen, fixed, n int) iter.Seq[int] {
	return GenFixedBy(g, fixed, func(g *Gen) int {
		return g.Pick(n)
	})
}

// GenPerm generates permutations of n elements. Yields indices in [0, n).
func GenPerm(g *Gen, n int) iter.Seq[int] {
	idxs := make([]int, n)
	for i := range n {
		idxs[i] = i
	}
	return func(yield func(int) bool) {
		remaining := make([]int, len(idxs))
		copy(remaining, idxs)
		for range n {
			picked := g.Gen(len(remaining) - 1)
			val := remaining[picked]
			remaining = append(remaining[:picked], remaining[picked+1:]...)
			if !yield(val) {
				// Drain remaining Gen calls to keep state consistent
				for range len(remaining) {
					g.Gen(len(remaining) - 1)
					remaining = remaining[:len(remaining)-1]
				}
				return
			}
		}
	}
}

// GenSubset generates subsets of n elements. Yields indices of included elements.
func GenSubset(g *Gen, n int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range n {
			if g.Flip() {
				if !yield(i) {
					// Drain remaining Flip calls to keep state consistent
					for j := i + 1; j < n; j++ {
						g.Flip()
					}
					return
				}
			}
		}
	}
}
