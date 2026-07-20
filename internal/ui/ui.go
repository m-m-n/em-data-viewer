package ui

import (
	"os"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// applicationID identifies this GApplication instance. It is also the
// Wayland app_id GNOME Shell uses to resolve the dock icon, and it MUST
// equal the basename of the shipped .desktop file (dpkg target in
// Makefile) for that resolution to succeed.
const applicationID = "io.github.m_m_n.EmDataViewer"

// osArgs is a seam over os.Args for testability (runArgs below).
var osArgs = os.Args

// runArgs returns the argument slice passed to the GTK application's Run
// call: only the program name (index 0). cmd/em-data-viewer's own CLI parsing
// is the single owner of the application-specific config-path argument
// (FR2); that argument must never reach GApplication, where it would be
// treated as a file to open and suppress the "activate" signal (AC-1).
func runArgs() []string {
	if len(osArgs) == 0 {
		return nil
	}
	return osArgs[:1]
}

// WindowSettings is the window-level configuration the app layer applies
// to the UI, translated from config.Config.Window into the ui package's
// own types so ui stays decoupled from the config package. There is no
// Height field (task0001): the window's height is derived from its
// content at runtime rather than configured.
type WindowSettings struct {
	Width int
	Style map[string]string
}

// UI is the GTK4 widget window: an undecorated frame with stacked
// sections, drag-to-move, a right-click context menu and a config-error
// banner. All methods assume they are called on the GTK main loop (the app
// layer guarantees this).
type UI struct {
	app   *gtk.Application
	quit  func()
	frame *frame

	// presented records whether Present has already taken effect — the
	// "first call wins" half of the PresenterUI contract
	// (IMPLEMENTATION.md Shared Components). presentPending records a
	// Present call that arrived before the frame existed, so activation
	// can honor it once the frame is built — the "remembered" half.
	presented      bool
	presentPending bool
}

// New constructs a UI. quit is invoked when the user chooses 終了 from the
// context menu; onKeepAboveToggle is invoked with the NEW desired value
// whenever the user toggles the「最前面に固定」menu check item (the caller
// routes this to SetKeepAbove and to persisting the value, per
// IMPLEMENTATION.md). Activation builds the frame but does not show it
// (FR1, task0002 AC-1): the window is presented only once, on request, via
// Present.
func New(quit func(), onKeepAboveToggle func(enabled bool)) *UI {
	u := &UI{
		app:  gtk.NewApplication(applicationID, gio.ApplicationFlagsNone),
		quit: quit,
	}
	u.app.ConnectActivate(func() {
		if u.frame != nil {
			// A repeat activation of this single-instance application (a
			// second launch) must not replace the existing frame: presented
			// is one-shot for the process lifetime, but recreating the frame
			// here would swap every subsequent ApplyWindow/SetSections/
			// UpdateSection call onto a new, hidden frame while the visible
			// window goes stale (task0003 Fix 3, finding 6ff3de2d6683a663).
			// Keep the existing frame; no re-registration either.
			return
		}
		// Registration must happen after activation begins (GTK is
		// initialized) and before newFrame builds the window, so the icon
		// theme has the resource path available the moment the window
		// requests windowIconName (task0001 Design; SPEC FR3).
		registerIconResources()
		u.frame = newFrame(u.app, u.requestQuit, onKeepAboveToggle)
		if u.presentPending {
			// A Present call arrived before this handler ran; honor it now
			// that the frame exists (task0002 AC-2).
			u.presentPending = false
			u.presented = true
			u.frame.present()
		}
	})
	return u
}

// Present shows the window for the first time, fitted to its final size so
// no visible resize follows presentation (FR5, US1). Only the first
// effective call presents; later calls are no-ops (IMPLEMENTATION.md
// Shared Components: PresenterUI contract). A call arriving before
// activation has built the GTK frame is remembered and honored once New's
// activate handler builds it — callers need not know which case applies
// (task0002 AC-1, AC-2).
func (u *UI) Present() {
	if u.presented {
		return
	}
	if u.frame == nil {
		u.presentPending = true
		return
	}
	u.presented = true
	u.frame.present()
}

// requestQuit runs the caller-supplied quit callback and terminates the
// GTK main loop.
func (u *UI) requestQuit() {
	if u.quit != nil {
		u.quit()
	}
	u.app.Quit()
}

// Run blocks on the GTK main loop until the application quits.
func (u *UI) Run() int {
	return u.app.Run(runArgs())
}

// ApplyWindow (re-)applies window-level settings to the existing window
// without recreating it (FR1).
func (u *UI) ApplyWindow(settings WindowSettings) {
	if u.frame == nil {
		return
	}
	u.frame.applySettings(settings)
}

// SetSections replaces the displayed sections with sections, in order
// (AC-4). A section add/remove can change the content's natural height, so
// this also re-fits the window height (task0001 Design: "Shrink
// requirement").
func (u *UI) SetSections(sections []SectionViewModel) {
	if u.frame == nil {
		return
	}
	u.frame.sections.setSections(sections)
	u.frame.restyle()
	u.frame.refit()
}

// UpdateSection replaces the content of exactly the section identified by
// key (AC-4). isError renders message in the error style with an
// "error: " prefix instead of treating it as pre-validated Pango markup
// (AC-6). A content change can change the section's natural height, so
// this also re-fits the window height (task0001 Design: "Shrink
// requirement").
func (u *UI) UpdateSection(key, markup string, isError bool) {
	if u.frame == nil {
		return
	}
	u.frame.sections.updateSection(key, markup, isError)
	u.frame.refit()
}

// SetConfigError shows the config-error banner above all sections. An
// empty message clears it (AC-5). Showing/hiding the banner changes the
// content's natural height, so this also re-fits the window height
// (task0001 Design: "Shrink requirement").
func (u *UI) SetConfigError(message string) {
	if u.frame == nil {
		return
	}
	u.frame.banner.setError(message)
	u.frame.refit()
}

// SetKeepAbove applies the keep-above state to the window: on X11 it sets
// (or clears) _NET_WM_STATE_ABOVE on the toplevel — deferred to the
// window's first realize when called before then; on Wayland it is a
// no-op apply, but the menu check mark still tracks the value
// (IMPLEMENTATION.md Shared Components; task plan AC-3, AC-4, AC-5).
func (u *UI) SetKeepAbove(enabled bool) {
	if u.frame == nil {
		return
	}
	u.frame.setKeepAbove(enabled)
}

// KeepAboveSupported reports whether the current backend can apply
// keep-above right now: true iff X11 (including XWayland) and the
// toplevel's XID is reachable (IMPLEMENTATION.md Shared Components).
func (u *UI) KeepAboveSupported() bool {
	if u.frame == nil {
		return false
	}
	return u.frame.keepAboveSupported()
}
