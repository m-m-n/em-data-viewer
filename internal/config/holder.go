package config

import "sync"

// Holder keeps the last successfully loaded Config plus the error
// from the most recent load attempt, if any. It gives three
// observable outcomes: an initial successful load; a successful
// reload, which replaces the held Config and clears any error; and a
// failed reload, which retains the previously held Config and
// records the new error (surfaced via Err until the next successful
// Update).
//
// Holder is the single authoritative source the app layer's reload path
// reads from (Rework Round 1: reload pipeline): it reads Config/Err back
// immediately after each Update rather than keeping a separate local copy
// of the load result, so Holder is never a write-only duplicate.
type Holder struct {
	mu     sync.RWMutex
	config Config
	err    error
}

// NewHolder returns an empty Holder: no config held yet and no
// error recorded.
func NewHolder() *Holder {
	return &Holder{}
}

// Update records the outcome of one load attempt (typically the
// result of config.Load). On success (err == nil) it replaces the
// held Config and clears any previously recorded error. On failure
// it keeps the previously held Config untouched and records err,
// which Err then returns until the next successful Update.
func (h *Holder) Update(cfg Config, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err != nil {
		h.err = err
		return
	}
	h.config = cfg
	h.err = nil
}

// Config returns the currently held (last good) config.
func (h *Holder) Config() Config {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.config
}

// Err returns the error from the most recent Update, or nil if the
// most recent Update succeeded (or none has happened yet).
func (h *Holder) Err() error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.err
}
