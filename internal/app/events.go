package app

import (
	"strings"

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

// consumeEvents reads runner events until events closes (Stop was called),
// translating each one and scheduling the resulting UI update via dispatch
// (glib.IdleAdd in production; a synchronous stand-in in tests) so no
// runner-goroutine code ever touches the UI directly (NFR2).
func consumeEvents(events <-chan runner.Event, snapshot func(key string) []string, u UI, dispatch func(func())) {
	for ev := range events {
		key, text, isError := translateEvent(ev, snapshot)
		dispatch(func() {
			u.UpdateSection(key, text, isError)
		})
	}
}
