// Package syncmap contains a simple generic wrapper around sync.Map.
// Hopefully to be deprecated in favor of a built-in equivalent in the future.
package syncmap

// A simple generic wrapper around sync.Map.
// Most of the code is from here:
// https://www.reddit.com/r/golang/comments/twucb0/is_there_already_a_generic_threadsafe_map/
// We used to use xsync.Map but ran into a deadlock bug in its LoadOrCompute function
// where a panic inside the compute function would never unlock a mutex, leading to a deadlock
// if another goroutine was trying to take the same lock.

import "sync"

type Map[K comparable, V any] struct {
	m sync.Map
}

func (m *Map[K, V]) Delete(key K) { m.m.Delete(key) }
func (m *Map[K, V]) Load(key K) (value V, ok bool) {
	v, ok := m.m.Load(key)
	if !ok {
		return value, ok
	}
	return v.(V), ok
}

func (m *Map[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	if !loaded {
		return value, loaded
	}
	return v.(V), loaded
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	a, loaded := m.m.LoadOrStore(key, value)
	return a.(V), loaded
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value any) bool { return f(key.(K), value.(V)) })
}
func (m *Map[K, V]) Store(key K, value V) { m.m.Store(key, value) }

// Follows the same interface as xsync.Map's LoadOrCompute. Compute should be idempotent.
func (m *Map[K, V]) LoadOrCompute(key K, compute func() (newValue V, cancel bool)) (V, bool) {
	// First check if the value exists
	if v, ok := m.Load(key); ok {
		return v, true
	}

	// Compute the new value
	newValue, cancel := compute()
	if cancel {
		var zero V
		return zero, false
	}

	// Try to store it, but another goroutine might have stored a value in the meantime
	actual, loaded := m.LoadOrStore(key, newValue)
	return actual, loaded
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{}
}
