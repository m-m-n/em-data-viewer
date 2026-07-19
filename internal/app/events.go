package app

import (
	"strings"
	"sync"

	"em-data-viewer/internal/markup"
	"em-data-viewer/internal/runner"
)

// translateEvent converts one runner.Event into the UI update it produces,
// GTK-free: replace and append content go through markup validation (FR6);
// error events map directly to the UI's error flag. snapshot supplies a
// stream section's current buffered lines for append events — the UI
// always receives a section's full current content, joined as lines,
// matching UpdateSection's "replace this section's displayed content"
// contract. All three kinds have trailing newlines stripped from the final
// display text before it reaches the UI (FR1/FR2/FR3): replace and append
// are stripped before markup validation/rendering, so markup handling is
// unaffected; the error message is stripped before the UI layer applies its
// "error: " prefix.
//
// Called only from inside consumeEvents' dispatched callback (Rework
// Round 1: render at dispatch time, AC-3) — never at receive time — so a
// coalesced-away intermediate event never pays for a render or a
// snapshot.
func translateEvent(ev runner.Event, snapshot func(key string) []string) (key, text string, isError bool) {
	switch ev.Kind {
	case runner.KindError:
		return ev.SectionKey, stripTrailingNewlines(ev.Text), true
	case runner.KindAppend:
		joined := stripTrailingNewlines(strings.Join(snapshot(ev.SectionKey), "\n"))
		rendered, _ := markup.Render(joined)
		return ev.SectionKey, rendered, false
	default: // runner.KindReplace
		rendered, _ := markup.Render(stripTrailingNewlines(ev.Text))
		return ev.SectionKey, rendered, false
	}
}

// stripTrailingNewlines removes newline characters from the end of text
// (FR1/FR2/FR3/FR4/FR5): a trailing "\n" is removed, and a "\r"
// immediately preceding a removed "\n" is removed along with it (SPEC Edge
// case 1). Interior newlines — including interior blank lines — and any
// carriage return not immediately followed by a removed newline are left
// byte-for-byte unchanged. Text made up only of newlines becomes the empty
// string.
func stripTrailingNewlines(text string) string {
	for {
		if strings.HasSuffix(text, "\r\n") {
			text = text[:len(text)-2]
			continue
		}
		if strings.HasSuffix(text, "\n") {
			text = text[:len(text)-1]
			continue
		}
		return text
	}
}

// sectionState is one section's last rendered UI content: the text and
// error flag most recently sent to UI.UpdateSection.
type sectionState struct {
	text    string
	isError bool
}

// sectionStore is the app-owned presentation store (Rework Round 1: reload
// pipeline & presentation state). It holds the latest rendered state per
// section key; every dispatched UI update below records it here, and the
// reload path (reload.go) reads it back to rehydrate widgets rebuilt by
// SetSections — restoring an interval section's last output and a failed
// section's terminal error display exactly, which the runner itself does
// not retain between runs (AC-1).
type sectionStore struct {
	mu    sync.Mutex
	state map[string]sectionState
}

// newSectionStore returns an empty sectionStore.
func newSectionStore() *sectionStore {
	return &sectionStore{state: make(map[string]sectionState)}
}

// set records key's latest rendered state.
func (s *sectionStore) set(key, text string, isError bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state[key] = sectionState{text: text, isError: isError}
}

// get returns key's latest rendered state, or (zero value, false) if
// nothing has ever been recorded for key.
func (s *sectionStore) get(key string) (sectionState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.state[key]
	return st, ok
}

// pendingSection holds one section's coalesced, not-yet-rendered event
// state: current is the raw event backing the already-scheduled dispatch
// callback; next, when non-nil, is a later event that arrived while
// current was an undisplayed error and therefore must not overwrite it
// (AC-2).
type pendingSection struct {
	current runner.Event
	next    *runner.Event
}

// consumeEvents reads runner events until events closes (Stop was called),
// scheduling at most one dispatch callback per section (glib.IdleAdd in
// production; a synchronous or capturing stand-in in tests) so no
// runner-goroutine code ever touches the UI directly (NFR2). Only the raw
// event is stored while pending; translateEvent (markup rendering, buffer
// snapshotting) runs inside the dispatched callback itself, never at
// receive time (AC-3).
//
// Coalescing: while a section's pending event has not yet been rendered, a
// newer event for the same section normally replaces it in place
// (steady-state coalescing keeps at most one pending callback per
// section) — UNLESS the not-yet-rendered event is an error, in which case
// the newer event is queued behind it instead of replacing it (at most one
// such preserved transition per section). This guarantees an error is
// rendered at least once before a later event for the same section takes
// its place (AC-2), while bounding the pending state to a small constant
// per section.
func consumeEvents(events <-chan runner.Event, snapshot func(key string) []string, u UI, store *sectionStore, dispatch func(func())) {
	var mu sync.Mutex
	pending := make(map[string]*pendingSection)

	var scheduleDispatch func(key string)
	scheduleDispatch = func(key string) {
		dispatch(func() {
			mu.Lock()
			p, ok := pending[key]
			if !ok {
				mu.Unlock()
				return
			}
			ev := p.current
			next := p.next
			if next != nil {
				p.current = *next
				p.next = nil
			} else {
				delete(pending, key)
			}
			mu.Unlock()

			renderedKey, text, isError := translateEvent(ev, snapshot)
			store.set(renderedKey, text, isError)
			u.UpdateSection(renderedKey, text, isError)

			if next != nil {
				// A newer event was preserved behind the one just
				// rendered (it arrived while that one was an undisplayed
				// error); schedule its own callback rather than
				// rendering it inline.
				scheduleDispatch(key)
			}
		})
	}

	for ev := range events {
		mu.Lock()
		p, has := pending[ev.SectionKey]
		if !has {
			pending[ev.SectionKey] = &pendingSection{current: ev}
			mu.Unlock()
			scheduleDispatch(ev.SectionKey)
			continue
		}
		if p.current.Kind == runner.KindError {
			next := ev
			p.next = &next
		} else {
			p.current = ev
		}
		mu.Unlock()
	}
}
