package runner

import "sync"

// RingBuffer is a fixed-capacity, thread-safe, in-memory buffer of lines.
// Appending beyond capacity drops the oldest line. Memory usage stays
// proportional to capacity regardless of how many lines are appended over
// the buffer's lifetime.
type RingBuffer struct {
	mu       sync.Mutex
	lines    []string // fixed-size ring, len(lines) == capacity
	start    int      // index of the oldest line
	count    int      // number of valid lines currently stored
	capacity int
}

// NewRingBuffer creates a RingBuffer holding at most capacity lines. A
// capacity below 1 is treated as 1.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity < 1 {
		capacity = 1
	}
	return &RingBuffer{
		lines:    make([]string, capacity),
		capacity: capacity,
	}
}

// Append adds line, dropping the oldest line first if the buffer is
// already at capacity.
func (b *RingBuffer) Append(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	idx := (b.start + b.count) % b.capacity
	if b.count < b.capacity {
		b.lines[idx] = line
		b.count++
		return
	}
	b.lines[b.start] = line
	b.start = (b.start + 1) % b.capacity
}

// Snapshot returns a copy of the current lines in order (oldest first).
func (b *RingBuffer) Snapshot() []string {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]string, b.count)
	for i := 0; i < b.count; i++ {
		out[i] = b.lines[(b.start+i)%b.capacity]
	}
	return out
}
