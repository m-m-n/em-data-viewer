package runner

import (
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestRunner_ApplyStartsStopsAndLeavesUnchangedRunning references AC-5:
// Apply starts added keys, stops removed keys, restarts changed keys, and
// leaves unchanged keys running. A stream section is used as the
// unchanged one so continuity (vs. a restart resetting it) is observable
// via strictly-increasing counter output, per the task's Test Notes.
func TestRunner_ApplyStartsStopsAndLeavesUnchangedRunning(t *testing.T) {
	r := New()
	defer r.Stop()

	counting := SectionConfig{
		Key:      "counter",
		Command:  `for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do echo $i; sleep 0.02; done`,
		Mode:     ModeStream,
		MaxLines: 20,
	}
	removedCfg := SectionConfig{
		Key:      "removed",
		Command:  "echo old",
		Mode:     ModeInterval,
		Interval: 15 * time.Millisecond,
	}

	r.Apply([]SectionConfig{counting, removedCfg})

	// Establish a solid counter baseline (reaching 2) before reconciling,
	// regardless of how "counter" and "removed" events interleave.
	var lastN int
	baselineDeadline := time.After(2 * time.Second)
	for lastN < 2 {
		select {
		case ev := <-r.Events():
			if ev.SectionKey == "counter" {
				if n, err := strconv.Atoi(ev.Text); err == nil {
					lastN = n
				}
			}
		case <-baselineDeadline:
			t.Fatal("timed out waiting for counter baseline before reconcile")
		}
	}

	addedCfg := SectionConfig{
		Key:      "added",
		Command:  "echo new",
		Mode:     ModeInterval,
		Interval: 15 * time.Millisecond,
	}
	// Reconcile: drop "removed", add "added", leave "counter" untouched.
	r.Apply([]SectionConfig{counting, addedCfg})

	var removedCount, addedCount int
	var counterAdvanced bool
	deadline := time.After(1500 * time.Millisecond)
collect:
	for {
		select {
		case ev := <-r.Events():
			switch ev.SectionKey {
			case "added":
				if ev.Kind == KindReplace && ev.Text == "new\n" {
					addedCount++
				}
			case "removed":
				removedCount++
			case "counter":
				if n, err := strconv.Atoi(ev.Text); err == nil && n > lastN {
					counterAdvanced = true
				}
			}
		case <-deadline:
			break collect
		}
	}

	if addedCount < 1 {
		t.Error("added section never emitted a replace event")
	}
	if !counterAdvanced {
		t.Errorf("unchanged stream section did not advance past %d; it may have been restarted or stalled", lastN)
	}
	if removedCount > 1 {
		t.Errorf("removed section kept producing events after reconcile: count=%d (want at most 1 straggler)", removedCount)
	}
}

// TestRunner_ErrorInOneSectionDoesNotAffectOthers references AC-4: a
// failing section emits error events for its own key only, while other
// sections keep running and producing their own events.
func TestRunner_ErrorInOneSectionDoesNotAffectOthers(t *testing.T) {
	r := New()
	defer r.Stop()

	r.Apply([]SectionConfig{
		{Key: "bad", Command: "exit 1", Mode: ModeInterval, Interval: 15 * time.Millisecond},
		{Key: "good", Command: "echo ok", Mode: ModeInterval, Interval: 15 * time.Millisecond},
	})

	var badErrors, goodReplaces int
	deadline := time.After(2 * time.Second)
	for badErrors < 2 || goodReplaces < 2 {
		select {
		case ev := <-r.Events():
			switch {
			case ev.SectionKey == "bad" && ev.Kind == KindError:
				badErrors++
			case ev.SectionKey == "bad":
				t.Errorf("bad section emitted unexpected kind %v", ev.Kind)
			case ev.SectionKey == "good" && ev.Kind == KindReplace:
				goodReplaces++
				if ev.Text != "ok\n" {
					t.Errorf("good section Text = %q, want %q", ev.Text, "ok\n")
				}
			case ev.SectionKey == "good":
				t.Errorf("good section emitted unexpected kind %v", ev.Kind)
			}
		case <-deadline:
			t.Fatalf("timed out: badErrors=%d goodReplaces=%d", badErrors, goodReplaces)
		}
	}
}

// TestRunner_StopTerminatesStubbornChildAndClosesChannel references
// AC-6: Stop terminates all child processes (a stubborn child is killed
// after the timeout) and closes the event channel.
func TestRunner_StopTerminatesStubbornChildAndClosesChannel(t *testing.T) {
	old := terminateGrace
	terminateGrace = 50 * time.Millisecond
	defer func() { terminateGrace = old }()

	r := New()
	r.Apply([]SectionConfig{
		{Key: "normal", Command: "echo hi", Mode: ModeInterval, Interval: 30 * time.Millisecond},
		{Key: "stubborn", Command: `trap '' TERM; while true; do sleep 0.05; done`, Mode: ModeStream, MaxLines: 10},
	})

	// Let both sections run for a bit before stopping.
	time.Sleep(150 * time.Millisecond)

	stopped := make(chan struct{})
	go func() {
		r.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("Stop() did not return in time; a stubborn child may not have been killed")
	}

	// Drain whatever was buffered before Stop closed the channel.
	for {
		_, ok := <-r.Events()
		if !ok {
			break
		}
	}

	// Further reads from the closed channel must return immediately with
	// the zero value.
	ev, ok := <-r.Events()
	if ok || ev != (Event{}) {
		t.Errorf("read from closed channel = %+v, ok=%v, want zero value, ok=false", ev, ok)
	}
}

// TestRunner_Snapshot exercises the Snapshot(key) accessor the Runner
// exposes for stream-mode sections, used by consumers that need the
// current buffered content (e.g. on a UI re-render).
func TestRunner_Snapshot(t *testing.T) {
	r := New()
	defer r.Stop()

	r.Apply([]SectionConfig{
		{Key: "log", Command: `printf 'x\ny\nz\n'`, Mode: ModeStream, MaxLines: 10},
	})

	seen := 0
	deadline := time.After(2 * time.Second)
	for seen < 3 {
		select {
		case ev := <-r.Events():
			if ev.SectionKey == "log" && ev.Kind == KindAppend {
				seen++
			}
		case <-deadline:
			t.Fatal("timed out waiting for log lines")
		}
	}

	got := r.Snapshot("log")
	want := []string{"x", "y", "z"}
	if len(got) != len(want) {
		t.Fatalf("Snapshot() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Snapshot()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if got := r.Snapshot("unknown"); got != nil {
		t.Errorf("Snapshot(unknown) = %#v, want nil", got)
	}
}

// TestNoGTKImport references AC-7: the package does not import gotk4 (so
// go test ./... passes headlessly).
func TestNoGTKImport(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	for _, pkg := range pkgs {
		for fname, file := range pkg.Files {
			for _, imp := range file.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				if strings.Contains(path, "gotk4") || strings.Contains(path, "/gtk") {
					t.Errorf("%s imports forbidden GUI package %q (NFR3: no gotk4 in internal/runner)", fname, path)
				}
			}
		}
	}
}
