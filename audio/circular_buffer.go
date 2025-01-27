package audio

import "sync"

// CircularBuf implements a circular buffer for tracking
type CircularBuf[T any] struct {
	mu   sync.RWMutex
	pl   []T
	head int
	tail int
	size uint
}

// NewCircularBuf constructs a new timestamped circular buffer
func NewCircularBuf[T any](capacity int) *CircularBuf[T] {
	return &CircularBuf[T]{
		pl:   make([]T, capacity),
		head: 0,
		tail: 0,
		size: 0,
	}
}

func (tc *CircularBuf[T]) zeroVal() (v T) {
	return
}

// Clear resets the circular buffer to an empty-state
func (tc *CircularBuf[T]) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	for z := range tc.pl {
		tc.pl[z] = tc.zeroVal()
	}
	tc.head = 0
	tc.tail = 0
	tc.size = 0
}

// Insert adds a new payload to this buffer (overwriting the oldest entry if
// necessary)
func (tc *CircularBuf[T]) Insert(p T) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.pl[tc.head] = p
	// If head == tail then the list was either full or empty. If full, then
	// we just overwrote the item at tail, so increment both (and don't
	// increment size).
	if tc.head == tc.tail && tc.size > 0 {
		tc.head = (tc.head + 1) % len(tc.pl)
		tc.tail = tc.head
	} else {
		// Otherwise, advance just the head pointer and the size.
		tc.head = (tc.head + 1) % len(tc.pl)
		tc.size++
	}
}

// Iter calls cb on every entry in the circular buffer
// Iteration starts with the oldest value and moves toward the most recent.
func (tc *CircularBuf[T]) Iter(cb func(tg *T)) {
	if tc == nil {
		return
	}
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if tc.size == 0 {
		return
	}
	ptr := tc.tail
	for i := uint(0); i < tc.size; i++ {
		cb(&tc.pl[ptr])
		ptr = (ptr + 1) % len(tc.pl)
	}
}

// Back returns the most recently inserted value in the circular buffer
func (tc *CircularBuf[T]) Back() (ret T, ok bool) {
	if tc == nil {
		return
	}
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if tc.size == 0 {
		return
	}
	// return the value immediately before head, unfortunately, if head is
	// 0, subtracting 1 gives us -1, and some mod implementations give us
	// -1 for that rather than len(tc.pl)-1, so add `len(tc.pl)` before
	// subtracting to guarantee a positive value.
	return tc.pl[(tc.head+len(tc.pl)-1)%len(tc.pl)], true
}

// Front returns the oldest value in the circular buffer.
func (tc *CircularBuf[T]) Front() (ret T, ok bool) {
	if tc == nil {
		return
	}
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if tc.size == 0 {
		return
	}

	return tc.pl[tc.tail], true
}

// PopFront returns and removes the oldest value in the circular buffer.
func (tc *CircularBuf[T]) PopFront() (ret T, ok bool) {
	if tc == nil {
		return
	}
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	if tc.size == 0 {
		return
	}

	ret = tc.pl[tc.tail]
	tc.pl[tc.tail] = tc.zeroVal()
	tc.tail = (tc.tail + 1) % len(tc.pl)
	tc.size--
	return ret, true
}

func (tc *CircularBuf[T]) Size() int {
	if tc == nil {
		return 0
	}
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	return int(tc.size)
}
