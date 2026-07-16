package runner

import (
	"context"
	"testing"
	"time"
)

// TestRunStream_EmitsAppendPerLine references AC-2: stream mode emits an
// append event per stdout line as it arrives, and that the runner-side
// buffer accumulates those lines (AC-3 cooperation).
func TestRunStream_EmitsAppendPerLine(t *testing.T) {
	events := make(chan Event, 10)
	buf := NewRingBuffer(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		runStream(ctx, "log", `printf 'one\ntwo\nthree\n'`, buf, events)
	}()

	want := []string{"one", "two", "three"}
	for i, w := range want {
		select {
		case ev := <-events:
			if ev.Kind != KindAppend {
				t.Fatalf("event %d: Kind = %v, want %v", i, ev.Kind, KindAppend)
			}
			if ev.SectionKey != "log" {
				t.Fatalf("event %d: SectionKey = %q, want %q", i, ev.SectionKey, "log")
			}
			if ev.Text != w {
				t.Fatalf("event %d: Text = %q, want %q", i, ev.Text, w)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for line %d", i)
		}
	}

	// The command has no more output and exits; expect an error event
	// with no restart (AC-4).
	select {
	case ev := <-events:
		if ev.Kind != KindError {
			t.Fatalf("Kind = %v, want %v", ev.Kind, KindError)
		}
		if ev.SectionKey != "log" {
			t.Fatalf("SectionKey = %q, want %q", ev.SectionKey, "log")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for exit error event")
	}

	<-done

	got := buf.Snapshot()
	if len(got) != len(want) {
		t.Fatalf("buffer Snapshot() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("buffer[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestRunStream_ProcessExitEmitsError references AC-4: a stream process
// exit (here: an immediately-exiting command) emits an error event
// carrying that section's key.
func TestRunStream_ProcessExitEmitsError(t *testing.T) {
	events := make(chan Event, 10)
	buf := NewRingBuffer(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		runStream(ctx, "quick", "true", buf, events)
	}()

	select {
	case ev := <-events:
		if ev.Kind != KindError {
			t.Fatalf("Kind = %v, want %v", ev.Kind, KindError)
		}
		if ev.SectionKey != "quick" {
			t.Fatalf("SectionKey = %q, want %q", ev.SectionKey, "quick")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for error event")
	}
	<-done
}

// TestRunStream_SpawnFailureEmitsError references AC-4: a spawn failure
// (unresolvable command) emits an error event rather than a silent
// return.
func TestRunStream_SpawnFailureEmitsError(t *testing.T) {
	events := make(chan Event, 10)
	buf := NewRingBuffer(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		runStream(ctx, "missing", "this-command-does-not-exist-xyz", buf, events)
	}()

	select {
	case ev := <-events:
		if ev.Kind != KindError {
			t.Fatalf("Kind = %v, want %v", ev.Kind, KindError)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for error event")
	}
	<-done
}

// TestRunStream_CtxCancelKillsStubbornChild references AC-6 (worker
// side): cancelling ctx terminates the process, falling back to a forced
// kill after the grace period when the child ignores the graceful
// signal, so runStream still returns promptly.
func TestRunStream_CtxCancelKillsStubbornChild(t *testing.T) {
	old := terminateGrace
	terminateGrace = 50 * time.Millisecond
	defer func() { terminateGrace = old }()

	events := make(chan Event, 10)
	buf := NewRingBuffer(10)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		runStream(ctx, "stubborn", `trap '' TERM; while true; do sleep 0.05; done`, buf, events)
	}()

	// Give the shell time to install the trap before we ask it to stop.
	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for runStream to return after killing a stubborn child")
	}
}
