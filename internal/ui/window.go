package ui

import (
	"log"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// minFrameHeight keeps the window grabbable even with zero sections
// (Design: "empty state (zero sections)").
const minFrameHeight = 48

// frame is the window chrome: an undecorated GtkApplicationWindow holding a
// vertical stack of the config-error banner and the sections view, with a
// live CSS provider, drag-to-move, the right-click context menu and the
// keep-above backend.
type frame struct {
	win      *gtk.ApplicationWindow
	root     *gtk.Box
	banner   *banner
	sections *sectionsView
	css      *gtk.CSSProvider

	windowStyle map[string]string

	keepAboveBackend keepAboveBackend
	keepAboveEnabled bool
	keepAboveMenu    *keepAboveMenu
}

// newFrame builds the undecorated frame window (FR1) and wires
// drag-to-move (AC-3), the right-click context menu (AC-7) and the
// keep-above backend (FR4). quit is invoked by the context menu's 終了
// item; onKeepAboveToggle is invoked with the NEW desired value whenever
// the user toggles the「最前面に固定」check item.
func newFrame(app *gtk.Application, quit func(), onKeepAboveToggle func(enabled bool)) *frame {
	win := gtk.NewApplicationWindow(app)
	win.SetDecorated(false)
	win.AddCSSClass("dv-window")
	win.SetSizeRequest(-1, minFrameHeight)

	root := gtk.NewBox(gtk.OrientationVertical, 0)
	win.SetChild(root)

	b := newBanner()
	root.Append(b.widget)

	sv := newSectionsView()
	root.Append(sv.widget)

	display := gdk.DisplayGetDefault()

	f := &frame{
		win:              win,
		root:             root,
		banner:           b,
		sections:         sv,
		css:              gtk.NewCSSProvider(),
		windowStyle:      MergeStyle(WindowStyleDefaults, nil),
		keepAboveBackend: selectKeepAboveBackend(display),
	}

	if display != nil {
		gtk.StyleContextAddProviderForDisplay(display, f.css, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	}
	f.restyle()

	attachDragToMove(win)

	// The label's Wayland hint is decided once from the (runtime-constant)
	// backend type; sensitivity/check mark are refreshed for real on
	// realize, once the toplevel's XID reachability is knowable (AC-3,
	// AC-4).
	initialState := deriveKeepAboveState(f.keepAboveEnabled, isX11Display(display))
	f.keepAboveMenu = attachContextMenu(win, quit, initialState, onKeepAboveToggle)

	win.ConnectRealize(func() {
		f.applyKeepAbove()
		f.refreshKeepAboveMenu()
	})

	return f
}

// setKeepAbove records the desired keep-above value and applies it: on X11
// this sets/clears _NET_WM_STATE_ABOVE on the toplevel immediately if
// realized, or is deferred to the first realize signal otherwise; on
// Wayland Apply is a no-op but the value — and therefore the menu check
// mark — is still tracked (IMPLEMENTATION.md Shared Components; AC-3,
// AC-4, AC-5).
func (f *frame) setKeepAbove(enabled bool) {
	f.keepAboveEnabled = enabled
	f.applyKeepAbove()
	f.refreshKeepAboveMenu()
}

// keepAboveSupported reports whether the backend can apply keep-above right
// now: true iff X11 (including XWayland) and the toplevel XID is reachable
// (IMPLEMENTATION.md Shared Components).
func (f *frame) keepAboveSupported() bool {
	if f.keepAboveBackend == nil {
		return false
	}
	surface := f.toplevelSurface()
	if surface == nil {
		return false
	}
	return f.keepAboveBackend.Supported(surface)
}

// applyKeepAbove pushes the tracked keep-above value to the backend. It is
// silently a no-op before the window is realized (no surface yet); the
// realize handler retries.
func (f *frame) applyKeepAbove() {
	if f.keepAboveBackend == nil {
		return
	}
	surface := f.toplevelSurface()
	if surface == nil {
		return
	}
	if err := f.keepAboveBackend.Apply(surface, f.keepAboveEnabled); err != nil {
		log.Printf("ui: keep-above: %v", err)
	}
}

// refreshKeepAboveMenu re-derives the menu item's check mark and
// sensitivity from the current tracked value and backend support (AC-5).
func (f *frame) refreshKeepAboveMenu() {
	f.keepAboveMenu.setState(deriveKeepAboveState(f.keepAboveEnabled, f.keepAboveSupported()))
}

// toplevelSurface returns the window's native surface, or nil before the
// window is realized.
func (f *frame) toplevelSurface() gdk.Surfacer {
	native := gtk.BaseWidget(f.win).Native()
	if native == nil {
		return nil
	}
	return native.Surface()
}

// applySettings applies window-level settings (size and style overrides)
// without recreating the window (FR1: styles are live).
func (f *frame) applySettings(settings WindowSettings) {
	if settings.Width > 0 && settings.Height > 0 {
		f.win.SetDefaultSize(settings.Width, settings.Height)
	}
	f.windowStyle = MergeStyle(WindowStyleDefaults, settings.Style)
	f.restyle()
}

// restyle rebuilds and reloads the full stylesheet (window + every current
// section) so style changes apply live without recreating widgets.
func (f *frame) restyle() {
	f.css.LoadFromString(buildStylesheet(f.windowStyle, f.sections.styleEntries()))
}

// attachDragToMove starts a compositor-side interactive move once a
// primary-button drag started anywhere on widget (including over child
// widgets such as section labels) exceeds GTK's standard drag threshold
// (FR1, AC-2; Wayland-compatible — never uses absolute positioning). The
// gesture is registered in the CAPTURE phase at the window level so child
// widgets never get a chance to swallow the press first, and a plain click
// (press+release with no qualifying movement) is never claimed, leaving it
// free to reach whatever child widget it landed on.
func attachDragToMove(widget gtk.Widgetter) {
	root := gtk.BaseWidget(widget)

	drag := gtk.NewGestureDrag()
	drag.SetButton(1) // primary (left) button only
	drag.SetPropagationPhase(gtk.PhaseCapture)

	var moveStarted bool
	drag.ConnectDragBegin(func(_, _ float64) {
		moveStarted = false
	})
	drag.ConnectDragUpdate(func(offsetX, offsetY float64) {
		if moveStarted {
			return
		}
		startX, startY, ok := drag.StartPoint()
		if !ok {
			return
		}
		if !root.DragCheckThreshold(int(startX), int(startY), int(startX+offsetX), int(startY+offsetY)) {
			return
		}
		moveStarted = true
		beginWindowMove(root, drag, startX, startY)
	})
	root.AddController(drag)
}

// beginWindowMove hands off to the compositor's interactive move for
// widget's toplevel, starting from the (x, y) point of widget where the
// drag began (Wayland-compatible — never uses absolute positioning).
func beginWindowMove(widget *gtk.Widget, gesture *gtk.GestureDrag, x, y float64) {
	native := widget.Native()
	if native == nil {
		return
	}
	toplevel, ok := native.Surface().(gdk.Topleveller)
	if !ok {
		return
	}
	nx, ny, ok := widget.TranslateCoordinates(native, x, y)
	if !ok {
		return
	}
	device := gesture.Device()
	if device == nil {
		return
	}
	var timestamp uint32
	if event := gesture.CurrentEvent(); event != nil {
		timestamp = gdk.BaseEvent(event).Time()
	}
	toplevel.BeginMove(device, int(gesture.CurrentButton()), nx, ny, timestamp)
}
