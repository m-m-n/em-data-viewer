package app

import (
	"strings"
	"sync"

	"data-viewer/internal/markup"
	"data-viewer/internal/runner"
)

// translateEvent converts one runner.Event into the UI update it produces
// (AC-4), GTK-free: replace and append content go through markup validation
// (FR6); error events map directly to the UI's error flag. snapshot
// supplies a stream section's current buffered lines for append events —
// the UI always receives a section's full current content, joined as
// lines, matching UpdateSection's "replace this section's displayed
// content" contract.
func translateEvent(ev runner.Event, snapshot func(key string) []string) (key, text string, isError bool) {
	switch ev.Kind {
	case runner.KindError:
		return ev.SectionKey, ev.Text, true
	case runner.KindAppend:
		rendered, _ := markup.Render(strings.Join(snapshot(ev.SectionKey), "\n"))
		return ev.SectionKey, rendered, false
	default: // runner.KindReplace
		rendered, _ := markup.Render(ev.Text)
		return ev.SectionKey, rendered, false
	}
}

// pendingUpdate holds the latest pending UI update for one section.
type pendingUpdate struct {
	text    string
	isError bool
}

// consumeEvents reads runner events until events closes (Stop was called),
// translating each one and scheduling the resulting UI update via dispatch
// (glib.IdleAdd in production; a synchronous stand-in in tests) so no
// runner-goroutine code ever touches the UI directly (NFR2).
//
// Updates are coalesced per section: at most one pending dispatch callback
// exists per section at a time. If events for a section arrive faster than
// dispatch can drain them, only the latest rendered snapshot is kept and no
// additional callback is scheduled, so the callback queue stays bounded and
// the runner channel's backpressure is preserved.
func consumeEvents(events <-chan runner.Event, snapshot func(key string) []string, u UI, dispatch func(func())) {
	var mu sync.Mutex
	pending := make(map[string]*pendingUpdate)

	for ev := range events {
		key, text, isError := translateEvent(ev, snapshot)

		mu.Lock()
		_, hasPending := pending[key]
		pending[key] = &pendingUpdate{text: text, isError: isError}
		mu.Unlock()

		if hasPending {
			// A callback for this section is already queued; it will pick
			// up the latest update we just stored. No need to schedule
			// another one.
			continue
		}

		dispatch(func() {
			mu.Lock()
			update := pending[key]
			delete(pending, key)
			mu.Unlock()

			if update == nil {
				return
			}
			u.UpdateSection(key, update.text, update.isError)
		})
	}
}
