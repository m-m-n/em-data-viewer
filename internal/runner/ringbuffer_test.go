package runner

import (
	"reflect"
	"testing"
)

// TestRingBuffer_Snapshot references AC-3: the ring buffer never exceeds
// max_lines, drops oldest first, and Snapshot returns current lines in
// order.
func TestRingBuffer_Snapshot(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
		appended []string
		want     []string
	}{
		{
			name:     "fewer lines than capacity keeps all in order",
			capacity: 5,
			appended: []string{"a", "b", "c"},
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "more lines than capacity drops oldest first",
			capacity: 3,
			appended: []string{"a", "b", "c", "d", "e"},
			want:     []string{"c", "d", "e"},
		},
		{
			name:     "empty buffer snapshot is empty",
			capacity: 3,
			appended: nil,
			want:     []string{},
		},
		{
			name:     "non-positive capacity is treated as 1",
			capacity: 0,
			appended: []string{"a", "b"},
			want:     []string{"b"},
		},
		{
			name:     "exactly at capacity keeps all lines",
			capacity: 3,
			appended: []string{"a", "b", "c"},
			want:     []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := NewRingBuffer(tt.capacity)
			for _, line := range tt.appended {
				rb.Append(line)
			}
			got := rb.Snapshot()

			effectiveCap := tt.capacity
			if effectiveCap < 1 {
				effectiveCap = 1
			}
			if len(got) > effectiveCap {
				t.Fatalf("snapshot length %d exceeds capacity %d", len(got), effectiveCap)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Snapshot() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestRingBuffer_SnapshotIsIndependentCopy guards against Snapshot
// exposing internal state that later Appends could mutate out from under
// a caller.
func TestRingBuffer_SnapshotIsIndependentCopy(t *testing.T) {
	rb := NewRingBuffer(2)
	rb.Append("a")
	snap := rb.Snapshot()

	rb.Append("b")
	rb.Append("c")

	if !reflect.DeepEqual(snap, []string{"a"}) {
		t.Errorf("earlier snapshot mutated: got %#v, want %#v", snap, []string{"a"})
	}

	got := rb.Snapshot()
	want := []string{"b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Snapshot() after further appends = %#v, want %#v", got, want)
	}
}
