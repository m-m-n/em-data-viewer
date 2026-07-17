package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestRunInterval_EmitsReplacePerRun references AC-1: interval mode emits
// a replace event with the command's full stdout for each run, at
// approximately the configured interval.
func TestRunInterval_EmitsReplacePerRun(t *testing.T) {
	const interval = 30 * time.Millisecond

	events := make(chan Event, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	start := time.Now()
	go func() {
		defer close(done)
		runInterval(ctx, "clock", "echo tick", interval, events)
	}()

	for i := 0; i < 3; i++ {
		select {
		case ev := <-events:
			if ev.Kind != KindReplace {
				t.Fatalf("event %d: Kind = %v, want %v", i, ev.Kind, KindReplace)
			}
			if ev.SectionKey != "clock" {
				t.Fatalf("event %d: SectionKey = %q, want %q", i, ev.SectionKey, "clock")
			}
			if ev.Text != "tick\n" {
				t.Fatalf("event %d: Text = %q, want %q", i, ev.Text, "tick\n")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for event %d", i)
		}
	}
	// The first run is immediate; two further runs each require a full
	// sleep of interval, so at least ~1 interval must have elapsed by now
	// (generous lower bound to catch a "runs without sleeping" bug without
	// being flaky about scheduler jitter).
	if elapsed := time.Since(start); elapsed < interval {
		t.Errorf("3 events arrived after %v, want at least ~%v (runs should be spaced by the interval)", elapsed, interval)
	}

	cancel()
	<-done
}

// TestRunInterval_FailureEmitsErrorAndContinues references AC-4: a
// failing command (non-zero exit) emits an error event for that section,
// and the interval cycle continues afterward.
func TestRunInterval_FailureEmitsErrorAndContinues(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{name: "non-zero exit", command: "exit 1"},
		{name: "command not found", command: "this-command-does-not-exist-xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := make(chan Event, 10)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			done := make(chan struct{})
			go func() {
				defer close(done)
				runInterval(ctx, "bad", tt.command, 20*time.Millisecond, events)
			}()

			for i := 0; i < 2; i++ {
				select {
				case ev := <-events:
					if ev.Kind != KindError {
						t.Fatalf("event %d: Kind = %v, want %v", i, ev.Kind, KindError)
					}
					if ev.SectionKey != "bad" {
						t.Fatalf("event %d: SectionKey = %q, want %q", i, ev.SectionKey, "bad")
					}
					if ev.Text == "" {
						t.Errorf("event %d: Text is empty, want a descriptive error message", i)
					}
				case <-time.After(2 * time.Second):
					t.Fatalf("timed out waiting for error event %d", i)
				}
			}

			cancel()
			<-done
		})
	}
}

// TestRunInterval_OutputOverflowEmitsErrorAndContinues references AC-3:
// a run whose stdout exceeds the capture limit is terminated (not left
// to grow memory unbounded), emits an error event noting the
// truncation for that section, and the interval cycle continues
// afterward (a later tick still produces events).
func TestRunInterval_OutputOverflowEmitsErrorAndContinues(t *testing.T) {
	oldLimit := intervalOutputLimit
	oldGrace := terminateGrace
	intervalOutputLimit = 64
	terminateGrace = 50 * time.Millisecond
	defer func() {
		intervalOutputLimit = oldLimit
		terminateGrace = oldGrace
	}()

	events := make(chan Event, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		runInterval(ctx, "noisy", "yes", 30*time.Millisecond, events)
	}()

	for i := 0; i < 2; i++ {
		select {
		case ev := <-events:
			if ev.Kind != KindError {
				t.Fatalf("event %d: Kind = %v, want %v", i, ev.Kind, KindError)
			}
			if ev.SectionKey != "noisy" {
				t.Fatalf("event %d: SectionKey = %q, want %q", i, ev.SectionKey, "noisy")
			}
			if !strings.Contains(ev.Text, "exceeded") {
				t.Errorf("event %d: Text = %q, want it to mention the output limit was exceeded", i, ev.Text)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for overflow error event %d (cycle should continue)", i)
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for runInterval to return after cancel (child should not be stranded)")
	}
}
