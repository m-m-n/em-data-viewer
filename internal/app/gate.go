package app

import (
	"sync"
	"time"
)

// startupReadyTimeout is the fixed startup-readiness timeout (FR3): if not
// every section armed on the gate has had its first data received within
// this duration of arming, the gate fires anyway. Not configurable in
// production; tests construct their own gate with a short timeout via
// newStartupGate so they never sleep-poll for the real one.
const startupReadyTimeout = 1 * time.Second

// startupGate decides when the window may first be shown (task plan
// Design): after every armed section key has been marked received, after
// its timeout elapses, or on an explicit immediate-fire request --
// whichever comes first -- firing its ready callback exactly once. GUI-free
// (NFR2): it knows nothing about GTK or the ui layer; ready is the only way
// it communicates outward, and the timeout is event-driven (time.AfterFunc),
// never polled (NFR1).
type startupGate struct {
	ready   func()
	timeout time.Duration

	mu       sync.Mutex
	started  bool // true once arm or fireNow has run for the first time
	fired    bool // true once ready has been invoked (or is in the process of being)
	owed     map[string]struct{}
	received map[string]struct{} // keys marked received before arm has run (task0003 Fix 1)
	timer    *time.Timer
}

// newStartupGate returns a gate that invokes ready exactly once. timeout is
// the duration arm waits, once armed with a non-empty key set, before
// firing on its own (FR3); production wiring passes startupReadyTimeout.
func newStartupGate(ready func(), timeout time.Duration) *startupGate {
	return &startupGate{
		ready:    ready,
		timeout:  timeout,
		owed:     make(map[string]struct{}),
		received: make(map[string]struct{}),
	}
}

// arm arms the gate with keys still owing their first data (Design step 1)
// and starts the timeout when keys is non-empty. Only the first call to arm
// or fireNow across the gate's lifetime has any effect (Design: "Arming is
// one-shot too"; IMPLEMENTATION.md decision 4: the gate applies to the
// first presentation only) -- every later call, including from a
// subsequent successful reload, is a no-op (AC-6). A key already recorded
// by a markReceived call that arrived before this arm call (task0003 Fix
// 1: the event source can start before the gate is armed) is reconciled
// out here instead of being added to owed, so arming with exactly that key
// set fires immediately and arming with that key plus others leaves only
// the others owed (AC-1). Arming with an empty owed set after reconciling
// fires the ready callback immediately (AC-5, FR4) instead of starting a
// timeout that would fire anyway.
func (g *startupGate) arm(keys []string) {
	g.mu.Lock()
	if g.started {
		g.mu.Unlock()
		return
	}
	g.started = true
	for _, k := range keys {
		if _, already := g.received[k]; already {
			continue
		}
		g.owed[k] = struct{}{}
	}
	g.received = nil // reconciliation done; no longer needed
	empty := len(g.owed) == 0
	if !empty {
		g.timer = time.AfterFunc(g.timeout, g.fire)
	}
	g.mu.Unlock()

	if empty {
		g.fire()
	}
}

// fireNow requests an immediate fire (AC-5, FR4: a first-load failure or a
// zero-section config). Like arm, only the first call to arm or fireNow
// across the gate's lifetime has any effect (AC-6).
func (g *startupGate) fireNow() {
	g.mu.Lock()
	if g.started {
		g.mu.Unlock()
		return
	}
	g.started = true
	g.mu.Unlock()

	g.fire()
}

// markReceived removes key from the owed set, regardless of the event kind
// that produced it -- callers pass the same key for replace, append and
// error events alike (FR2, AC-2, AC-3); an unknown or already-received key
// is ignored. Fires the ready callback the moment this empties the owed set
// (AC-1).
//
// A call arriving before arm has ever run (task0003 Fix 1: in production
// the event source can start emitting before the gate is armed) is
// buffered in received instead of being discarded: it never touches owed
// and never fires the callback itself (AC-2) -- arm reconciles it against
// the key set it is called with once it runs.
func (g *startupGate) markReceived(key string) {
	g.mu.Lock()
	if !g.started {
		g.received[key] = struct{}{}
		g.mu.Unlock()
		return
	}
	if _, ok := g.owed[key]; !ok {
		g.mu.Unlock()
		return
	}
	delete(g.owed, key)
	empty := len(g.owed) == 0
	g.mu.Unlock()

	if empty {
		g.fire()
	}
}

// fire invokes ready exactly once (AC-6): the mutex-guarded fired flag
// makes this safe when the timeout and the last markReceived race each
// other -- exactly one of them observes fired == false and proceeds -- and
// stopping any pending timer here means a timeout that loses the race can
// never fire a second time later.
func (g *startupGate) fire() {
	g.mu.Lock()
	if g.fired {
		g.mu.Unlock()
		return
	}
	g.fired = true
	if g.timer != nil {
		g.timer.Stop()
	}
	g.mu.Unlock()

	if g.ready != nil {
		g.ready()
	}
}
