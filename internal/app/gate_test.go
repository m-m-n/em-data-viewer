package app

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"em-data-viewer/internal/config"
	"em-data-viewer/internal/runner"
)

// --- AC-1: fires exactly once when every armed key is received ---

// TestStartupGate_FiresExactlyOnceWhenAllArmedKeysReceived references AC-1.
func TestStartupGate_FiresExactlyOnceWhenAllArmedKeysReceived(t *testing.T) {
	var readyCalls int
	g := newStartupGate(func() { readyCalls++ }, time.Hour)

	g.arm([]string{"a", "b", "c"})
	g.markReceived("a")
	if readyCalls != 0 {
		t.Fatalf("ready calls = %d after marking one of three keys, want 0", readyCalls)
	}
	g.markReceived("b")
	if readyCalls != 0 {
		t.Fatalf("ready calls = %d after marking two of three keys, want 0", readyCalls)
	}
	g.markReceived("c")
	if readyCalls != 1 {
		t.Fatalf("ready calls = %d after marking all three keys, want 1", readyCalls)
	}
}

// TestStartupGate_MarkReceivedIgnoresUnknownKey covers markReceived's
// "unknown keys are ignored" clause (Design).
func TestStartupGate_MarkReceivedIgnoresUnknownKey(t *testing.T) {
	var readyCalls int
	g := newStartupGate(func() { readyCalls++ }, time.Hour)

	g.arm([]string{"a"})
	g.markReceived("never-armed")

	if readyCalls != 0 {
		t.Fatalf("ready calls = %d after marking an unarmed key, want 0 (unknown keys are ignored, \"a\" still owed)", readyCalls)
	}
}

// --- AC-4: fires on timeout with keys still owed ---

// TestStartupGate_FiresWhenTimeoutElapsesWithKeysStillOwed references AC-4.
func TestStartupGate_FiresWhenTimeoutElapsesWithKeysStillOwed(t *testing.T) {
	fired := make(chan struct{})
	g := newStartupGate(func() { close(fired) }, 5*time.Millisecond)

	g.arm([]string{"a", "b"})
	g.markReceived("a") // "b" is still owed when the timeout elapses

	select {
	case <-fired:
	case <-time.After(time.Second):
		t.Fatal("ready was never invoked by the timeout")
	}
}

// --- AC-5: zero-key arm and explicit fireNow both fire without waiting ---

// TestStartupGate_ArmWithNoKeysFiresImmediately references AC-5.
func TestStartupGate_ArmWithNoKeysFiresImmediately(t *testing.T) {
	var readyCalls int
	// A long timeout proves the fire came from the empty key set, not a
	// timeout race.
	g := newStartupGate(func() { readyCalls++ }, time.Hour)

	g.arm(nil)

	if readyCalls != 1 {
		t.Fatalf("ready calls = %d immediately after arm(nil), want 1 (nothing to wait for)", readyCalls)
	}
}

// TestStartupGate_FireNowFiresImmediately references AC-5's explicit
// immediate-fire request (FR4: first-load failure).
func TestStartupGate_FireNowFiresImmediately(t *testing.T) {
	var readyCalls int
	g := newStartupGate(func() { readyCalls++ }, time.Hour)

	g.fireNow()

	if readyCalls != 1 {
		t.Fatalf("ready calls = %d immediately after fireNow(), want 1", readyCalls)
	}
}

// --- AC-6: one-shot semantics ---

// TestStartupGate_FiresOnlyOnceIgnoringLaterTriggers references AC-6: after
// the callback has fired once, further marks, fireNow requests, and re-arm
// attempts never fire it again.
func TestStartupGate_FiresOnlyOnceIgnoringLaterTriggers(t *testing.T) {
	var readyCalls int
	g := newStartupGate(func() { readyCalls++ }, time.Hour)

	g.arm([]string{"a"})
	g.markReceived("a")
	if readyCalls != 1 {
		t.Fatalf("ready calls = %d after the arming key was received, want 1", readyCalls)
	}

	g.markReceived("a")  // already removed; must stay a no-op
	g.fireNow()          // gate already started; must be a no-op
	g.arm([]string{"b"}) // re-arm attempt; must be a no-op

	if readyCalls != 1 {
		t.Fatalf("ready calls = %d after further marks/fireNow/re-arm, want still 1", readyCalls)
	}

	// The re-arm attempt above must not have actually armed "b": marking it
	// received must not fire a second time either.
	g.markReceived("b")
	if readyCalls != 1 {
		t.Fatalf("ready calls = %d after marking the re-arm attempt's key, want still 1 (re-arm must be a full no-op)", readyCalls)
	}
}

// TestStartupGate_ConcurrentTimeoutAndLastMarkReceivedFireExactlyOnce
// references AC-6's concurrency corollary (Test Notes): the timeout firing
// concurrently with the last markReceived call must still invoke ready
// exactly once.
func TestStartupGate_ConcurrentTimeoutAndLastMarkReceivedFireExactlyOnce(t *testing.T) {
	var count int32
	done := make(chan struct{})
	g := newStartupGate(func() {
		atomic.AddInt32(&count, 1)
		close(done) // panics if ready is ever invoked a second time
	}, 2*time.Millisecond)

	g.arm([]string{"a"})
	go g.markReceived("a") // races the timeout goroutine

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ready was never invoked")
	}
	// Give a would-be second invocation time to occur; close(done) above
	// would already have panicked the test if it happened.
	time.Sleep(20 * time.Millisecond)
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Fatalf("ready invoked %d times, want exactly 1", got)
	}
}

// --- AC-2/AC-3: events.go wiring is kind-agnostic (consumeEvents -> gate) ---

// TestConsumeEvents_ErrorEventMarksSectionReceivedOnGate references AC-2: an
// error event's key counts as received, exactly like a success event's.
func TestConsumeEvents_ErrorEventMarksSectionReceivedOnGate(t *testing.T) {
	events := make(chan runner.Event, 1)
	events <- runner.Event{SectionKey: "a", Kind: runner.KindError, Text: "boom"}
	close(events)

	u := &fakeUI{}
	store := newSectionStore()

	var readyCalls int
	g := newStartupGate(func() { readyCalls++ }, time.Hour)
	g.arm([]string{"a"})

	consumeEvents(events, func(string) []string { return nil }, u, store, syncDispatch, g.markReceived)

	if readyCalls != 1 {
		t.Fatalf("gate ready calls = %d, want 1 (an error event must count as the section's first data)", readyCalls)
	}
}

// TestConsumeEvents_StreamAppendEventMarksSectionReceivedOnGate references
// AC-3: the gate is kind-agnostic -- a stream section's first append event
// marks it received.
func TestConsumeEvents_StreamAppendEventMarksSectionReceivedOnGate(t *testing.T) {
	events := make(chan runner.Event, 1)
	events <- runner.Event{SectionKey: "s", Kind: runner.KindAppend, Text: "line one"}
	close(events)

	u := &fakeUI{}
	store := newSectionStore()

	var readyCalls int
	g := newStartupGate(func() { readyCalls++ }, time.Hour)
	g.arm([]string{"s"})

	consumeEvents(events, func(string) []string { return nil }, u, store, syncDispatch, g.markReceived)

	if readyCalls != 1 {
		t.Fatalf("gate ready calls = %d, want 1 (an append event must count as the section's first data)", readyCalls)
	}
}

// --- AC-7: FIFO dispatch ordering ---

// TestConsumeEvents_GateReadyDispatchedAfterCompletingSectionsUpdateWasScheduled
// references AC-7: the present request is dispatched through the main-loop
// dispatch seam strictly after the gate-completing section event's UI
// update was scheduled, so a FIFO dispatcher applies that update first.
func TestConsumeEvents_GateReadyDispatchedAfterCompletingSectionsUpdateWasScheduled(t *testing.T) {
	events := make(chan runner.Event, 1)
	events <- runner.Event{SectionKey: "a", Kind: runner.KindReplace, Text: "hello"}
	close(events)

	u := &fakeUI{}
	store := newSectionStore()
	capture := &capturingDispatch{}

	presentDispatched := false
	// A long timeout proves the fire came from markReceived, not a race
	// with the timer.
	g := newStartupGate(func() {
		capture.dispatch(func() { presentDispatched = true })
	}, time.Hour)
	g.arm([]string{"a"})

	consumeEvents(events, func(string) []string { return nil }, u, store, capture.dispatch, g.markReceived)

	if len(capture.calls) != 2 {
		t.Fatalf("dispatch calls = %d, want 2 (the section update, then the present request)", len(capture.calls))
	}

	capture.calls[0]()
	if len(u.updates) != 1 {
		t.Fatalf("running the FIRST dispatched call did not apply the section update -- want the update queued before the present request; updates = %+v", u.updates)
	}
	if presentDispatched {
		t.Fatal("present was already dispatched after running only the first queued call -- want it queued strictly after the update")
	}

	capture.calls[1]()
	if !presentDispatched {
		t.Fatal("running the SECOND dispatched call did not trigger the present request")
	}
}

// --- AC-8: presenter seam (applyPresent) ---

// presentRecorder stands in for *ui.UI once it implements PresenterUI
// (task0002); it embeds fakeUI (defined in app_test.go) so it also
// satisfies UI.
type presentRecorder struct {
	fakeUI
	presentCalls int
}

func (p *presentRecorder) Present() { p.presentCalls++ }

// TestApplyPresent_CallsPresentWhenImplemented references AC-8's positive
// case.
func TestApplyPresent_CallsPresentWhenImplemented(t *testing.T) {
	u := &presentRecorder{}

	applyPresent(u)

	if u.presentCalls != 1 {
		t.Fatalf("presentCalls = %d, want 1", u.presentCalls)
	}
}

// TestApplyPresent_NoOpWhenNotImplemented references AC-8's negative case:
// *ui.UI does not implement PresenterUI in this worktree yet (task0002 has
// not merged), and applyPresent must not panic or otherwise fail.
func TestApplyPresent_NoOpWhenNotImplemented(t *testing.T) {
	u := &fakeUI{}

	applyPresent(u) // must not panic
}

// --- reload.go wiring: arm on first success, fireNow on first failure,
// one-shot across reloads (task plan Scope: "on the FIRST successful config
// application, arm the gate ...; on a first-load failure or a zero-section
// config, tell the gate to fire immediately") ---

// TestApplyConfig_SuccessArmsGateWithSectionKeysOnly verifies a successful
// reload arms the gate with exactly its own (non-break) section keys: an
// unrelated key never completes the gate, but both of cfg's section keys
// together do.
func TestApplyConfig_SuccessArmsGateWithSectionKeysOnly(t *testing.T) {
	cfg := sampleConfig()
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	var readyCalls int
	g := newStartupGate(func() { readyCalls++ }, time.Hour)

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, syncDispatch, g)

	g.markReceived("some-unrelated-key")
	if readyCalls != 0 {
		t.Fatalf("ready calls = %d after marking an unrelated key, want 0 (gate armed only with cfg's own section keys)", readyCalls)
	}

	g.markReceived(cfg.Sections[0].Key())
	g.markReceived(cfg.Sections[1].Key())
	if readyCalls != 1 {
		t.Fatalf("ready calls = %d after marking both of cfg's section keys received, want 1", readyCalls)
	}
}

// TestApplyConfig_ZeroSectionSuccessFiresGateImmediately covers the
// zero-section success case (FR4).
func TestApplyConfig_ZeroSectionSuccessFiresGateImmediately(t *testing.T) {
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	var readyCalls int
	g := newStartupGate(func() { readyCalls++ }, time.Hour)

	applyConfig(config.Config{}, nil, r, store, gens.next(), gens, u, syncDispatch, g)

	if readyCalls != 1 {
		t.Fatalf("ready calls = %d, want 1 (a zero-section success has nothing to wait for)", readyCalls)
	}
}

// TestApplyConfig_LoadFailureFiresGateImmediately covers the first-load
// failure case (FR4).
func TestApplyConfig_LoadFailureFiresGateImmediately(t *testing.T) {
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	var readyCalls int
	g := newStartupGate(func() { readyCalls++ }, time.Hour)

	applyConfig(config.Config{}, errors.New("boom"), r, store, gens.next(), gens, u, syncDispatch, g)

	if readyCalls != 1 {
		t.Fatalf("ready calls = %d, want 1 (a first-load failure has nothing to wait for)", readyCalls)
	}
}

// TestApplyConfig_SecondSuccessfulReloadDoesNotReArmOrRefireGate references
// IMPLEMENTATION.md decision 4: the gate applies to the first presentation
// only -- a later reload must not re-arm or re-fire it.
func TestApplyConfig_SecondSuccessfulReloadDoesNotReArmOrRefireGate(t *testing.T) {
	cfg := sampleConfig()
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	var readyCalls int
	g := newStartupGate(func() { readyCalls++ }, time.Hour)

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, syncDispatch, g)
	g.markReceived(cfg.Sections[0].Key())
	g.markReceived(cfg.Sections[1].Key())
	if readyCalls != 1 {
		t.Fatalf("ready calls after the first reload's keys were received = %d, want 1", readyCalls)
	}

	// A second successful reload, with a different section set, must not
	// re-arm (which would otherwise wait on new keys) or re-fire.
	applyConfig(sampleConfigWithBreak(), nil, r, store, gens.next(), gens, u, syncDispatch, g)
	if readyCalls != 1 {
		t.Fatalf("ready calls after a second successful reload = %d, want still 1 (no re-arm/re-fire)", readyCalls)
	}
}

// TestApplyConfig_ZeroSectionSuccess_GateFiresAfterProjectionDispatchQueued
// covers the same FIFO ordering guarantee as AC-7, but for the immediate-fire
// path taken by applyConfig itself (Design step 1): the present request
// must be queued strictly after the reload's own config-projection dispatch
// call, so a FIFO dispatcher applies the window/sections before presenting.
func TestApplyConfig_ZeroSectionSuccess_GateFiresAfterProjectionDispatchQueued(t *testing.T) {
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()
	capture := &capturingDispatch{}

	presentDispatched := false
	g := newStartupGate(func() {
		capture.dispatch(func() { presentDispatched = true })
	}, time.Hour)

	applyConfig(config.Config{}, nil, r, store, gens.next(), gens, u, capture.dispatch, g)

	if len(capture.calls) != 2 {
		t.Fatalf("dispatch calls = %d, want 2 (the config projection, then the present request)", len(capture.calls))
	}

	capture.calls[0]()
	if len(u.windowSettings) != 1 {
		t.Fatal("running the FIRST dispatched call did not apply the config projection, want it queued before the present request")
	}
	if presentDispatched {
		t.Fatal("present was already dispatched before the config projection's dispatch call ran")
	}

	capture.calls[1]()
	if !presentDispatched {
		t.Fatal("running the SECOND dispatched call did not trigger the present request")
	}
}
