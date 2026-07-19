// Package config implements the data-viewer YAML configuration:
// schema types, loading with defaults and validation, a last-good
// config holder, and a debounced file watcher. This package must not
// import gotk4 or anything GTK-related (NFR3) so it stays testable
// headlessly with `go test ./...`.
package config

import (
	"fmt"
	"time"
)

// Section modes.
const (
	ModeInterval = "interval"
	ModeStream   = "stream"
)

// TypeBreak is the YAML `type` value that marks a section entry as a
// forced row-break (section-row-break task0001 FR1) rather than a normal
// section.
const TypeBreak = "break"

// Defaults applied when the corresponding YAML field is omitted.
const (
	DefaultWindowWidth = 400
	DefaultMaxLines    = 100
	DefaultKeepAbove   = true
)

// Config is the parsed and validated application configuration.
// A Config with zero Sections is valid (empty, frame-only window).
type Config struct {
	Window   Window
	Sections []SectionConfig
}

// Window holds the widget window's width, style, and keep-above
// settings. Height is not configurable (task0001): the window's height
// is derived from its content at runtime by the ui layer.
type Window struct {
	Width     int
	Style     map[string]string
	KeepAbove bool
}

// SectionConfig defines one output section: the command it runs, how
// it runs (interval | stream), and its display style. An entry with
// IsBreak true is a forced row-break marker (`- type: break` in YAML,
// section-row-break task0001 FR1): every other field is zero-valued and
// meaningless for it. Downstream layers must exclude break entries from
// duplicate-key validation, runner reconciliation, and rehydration, while
// preserving their position in Sections so the ui layer can use them to
// force a row boundary at that point.
type SectionConfig struct {
	Name     string
	Command  string
	Mode     string
	Interval time.Duration
	MaxLines int
	Style    map[string]string
	IsBreak  bool
}

// Key returns a stable identity for the section derived from Name,
// Command, Mode, Interval, and MaxLines. Name is included so that two
// sections with identical execution settings but different names are
// treated as distinct sections; renaming a section therefore restarts
// its process. Style is deliberately excluded so style-only config
// edits keep the same key (FR3): reconciliation treats a section with
// an unchanged Key as unchanged.
func (s SectionConfig) Key() string {
	return fmt.Sprintf("%s\x00%s\x00%s\x00%d\x00%d", s.Name, s.Command, s.Mode, s.Interval, s.MaxLines)
}
