package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

// keepAboveWaylandHint explains, in the check item's own label, why it is
// insensitive on a backend that cannot apply keep-above (SPEC FR4: "a hint
// that the WM's 'Always on Top' (Alt+Space) can be used").
const keepAboveWaylandHint = "ウィンドウマネージャーの機能（例: Alt+Space）をご利用ください"

// keepAboveState is the pure, GTK-free decision for the「最前面に固定」menu
// item: whether it shows a check mark, whether it is clickable, and — when
// the backend cannot honor it — the hint text explaining why (SPEC FR3/FR4;
// AC-1, AC-4, AC-5).
type keepAboveState struct {
	Checked   bool
	Sensitive bool
	Hint      string
}

// deriveKeepAboveState computes the menu item's check mark, sensitivity and
// hint from whether keep-above is currently enabled and whether the backend
// can apply it right now (AC-1). The check mark always reflects the tracked
// enabled value regardless of backend support (IMPLEMENTATION.md: "config
// stores intent regardless of backend so switching backends later honors
// the setting"), so it keeps tracking the desired state even when the item
// is insensitive (AC-4, AC-5).
func deriveKeepAboveState(enabled, supported bool) keepAboveState {
	state := keepAboveState{Checked: enabled, Sensitive: supported}
	if !supported {
		state.Hint = keepAboveWaylandHint
	}
	return state
}

// keepAboveBackend applies or removes the platform's "always on top" hint
// for a toplevel surface. Isolated behind this interface so the rest of ui
// stays backend-agnostic and testable without X11/Wayland (SPEC NFR2); the
// only implementation living outside topmost_x11.go is the inert one below.
type keepAboveBackend interface {
	// Supported reports whether this backend can currently apply keep-above
	// for surface (e.g. X11 with a reachable XID).
	Supported(surface gdk.Surfacer) bool
	// Apply adds (above=true) or removes (above=false) the backend's
	// keep-above state for surface's toplevel.
	Apply(surface gdk.Surfacer, above bool) error
}

// inertKeepAboveBackend is used whenever the display isn't X11 (native
// Wayland) or can't be classified: keep-above is unsupported and Apply is a
// no-op, so no X11 calls are ever made (SPEC FR4, AC-4).
type inertKeepAboveBackend struct{}

func (inertKeepAboveBackend) Supported(gdk.Surfacer) bool    { return false }
func (inertKeepAboveBackend) Apply(gdk.Surfacer, bool) error { return nil }

// selectKeepAboveBackend returns the backend implementation for display's
// concrete GDK backend (SPEC FR4: "Backend detection at runtime from the GDK
// display type"). A nil or unrecognized display degrades to inert.
func selectKeepAboveBackend(display *gdk.Display) keepAboveBackend {
	if isX11Display(display) {
		return newX11KeepAboveBackend()
	}
	return inertKeepAboveBackend{}
}
