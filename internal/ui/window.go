package ui

import (
	"log"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
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
	attachReflowTriggers(win, f)

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

// attachReflowTriggers wires the window's default-width property notify and
// map signal to a deferred row-repack-plus-height-refit of the frame
// (Design: Triggers; FR4; task0001 Design: "Shrink requirement"). Each
// defers by one main-loop iteration via glib.IdleAdd so the sections
// container's allocation has settled to the new size by the time reflow
// measures it.
func attachReflowTriggers(win *gtk.ApplicationWindow, f *frame) {
	win.NotifyProperty("default-width", func() {
		glib.IdleAdd(f.reflowAndRefit)
	})
	win.ConnectMap(func() {
		glib.IdleAdd(f.reflowAndRefit)
	})
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

// applySettings applies window-level settings (width and style overrides)
// without recreating the window (FR1: styles are live). There is no
// configurable height (task0001): the height is derived from content by
// refit below. A style change can change the content's natural height
// (e.g. a font-size change), so this also re-fits the window height.
func (f *frame) applySettings(settings WindowSettings) {
	if settings.Width > 0 {
		f.win.SetDefaultSize(settings.Width, -1)
	}
	f.windowStyle = MergeStyle(WindowStyleDefaults, settings.Style)
	f.restyle()
	f.refit()
}

// restyle rebuilds and reloads the full stylesheet (window + every current
// section) so style changes apply live without recreating widgets.
func (f *frame) restyle() {
	f.css.LoadFromString(buildStylesheet(f.windowStyle, f.sections.styleEntries()))
}

// reflowAndRefit repacks the sections view's rows for the window's current
// width, then re-fits the window height to the result (task0001 Design:
// "Shrink requirement" re-fit triggers driven by a width change or the
// window becoming mapped).
func (f *frame) reflowAndRefit() {
	f.sections.reflow()
	f.refit()
}

// refit resizes the window's height to fit the root box's current natural
// content height (banner + sections), capped at the monitor height once the
// window is realized (task0001 Design: "Shrink requirement", "Monitor
// cap"). Width is left untouched. The minimum frame height (NFR2) is
// additionally enforced as a hard floor by win.SetSizeRequest in newFrame;
// clamping to it here too keeps the computed default size consistent with
// that floor.
func (f *frame) refit() {
	_, natural, _, _ := f.root.Measure(gtk.OrientationVertical, f.root.AllocatedWidth())
	height := fitHeight(natural, minFrameHeight, f.monitorHeight())

	width, _ := f.win.DefaultSize()
	f.win.SetDefaultSize(width, height)
}

// monitorHeight returns the height, in pixels, of the monitor the realized
// window's surface currently sits on, or 0 before the window is realized
// (no monitor known yet, so refit applies no cap; task0001 Design:
// "Monitor cap").
func (f *frame) monitorHeight() int {
	surface := f.toplevelSurface()
	if surface == nil {
		return 0
	}
	display := gdk.DisplayGetDefault()
	if display == nil {
		return 0
	}
	monitor := display.MonitorAtSurface(surface)
	if monitor == nil {
		return 0
	}
	return monitor.Geometry().Height()
}

// fitHeight computes the window height to apply, given the content's
// current natural height, the minimum frame height (NFR2, so a
// zero-section window stays grabbable), and the monitor height cap (0 =
// monitor not yet known, so no cap is applied; task0001 Design: "Monitor
// cap"). Pure and GUI-free so it is unit-testable independently of GTK
// measurement/monitor lookups (task0001 Test Notes).
func fitHeight(contentNatural, minHeight, monitorHeight int) int {
	height := contentNatural
	if height < minHeight {
		height = minHeight
	}
	if monitorHeight > 0 && height > monitorHeight {
		height = monitorHeight
	}
	return height
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
	surface := native.Surface()
	if surface == nil {
		return
	}
	// The backend-specific surface GTypes (GdkWaylandToplevel /
	// GdkX11Toplevel) are not registered with gotk4, so the dynamic wrapper
	// falls back to the base *gdk.Surface and a type assertion to
	// gdk.Topleveller always fails. Wrap the same object as a Toplevel
	// explicitly — a realized top-level window's surface always implements
	// GdkToplevel on every backend.
	toplevel := gdk.Toplevel{Surface: *gdk.BaseSurface(surface)}
	nx, ny, ok := widget.TranslateCoordinates(native, x, y)
	if !ok {
		return
	}
	device := gesture.Device()
	if device == nil {
		return
	}
	// CurrentEvent() resolves through gotk4's event-wrapper registry and can
	// come back nil for backend event types, yielding timestamp 0 — which
	// compositors ignore for interactive moves. CurrentEventTime() reads the
	// timestamp directly from the controller instead.
	toplevel.BeginMove(device, int(gesture.CurrentButton()), nx, ny, gesture.CurrentEventTime())
}
