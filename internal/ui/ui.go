package ui

import (
	"os"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// applicationID identifies this GApplication instance. It has no meaning
// beyond satisfying GApplication's identifier requirement.
const applicationID = "org.dataviewer.Widget"

// osArgs is a seam over os.Args for testability (runArgs below).
var osArgs = os.Args

// runArgs returns the argument slice passed to the GTK application's Run
// call: only the program name (index 0). cmd/data-viewer's own CLI parsing
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
}

// New constructs a UI. quit is invoked when the user chooses 終了 from the
// context menu; onKeepAboveToggle is invoked with the NEW desired value
// whenever the user toggles the「最前面に固定」menu check item (the caller
// routes this to SetKeepAbove and to persisting the value, per
// IMPLEMENTATION.md).
func New(quit func(), onKeepAboveToggle func(enabled bool)) *UI {
	u := &UI{
		app:  gtk.NewApplication(applicationID, gio.ApplicationFlagsNone),
		quit: quit,
	}
	u.app.ConnectActivate(func() {
		u.frame = newFrame(u.app, u.requestQuit, onKeepAboveToggle)
		u.frame.win.Present()
	})
	return u
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
