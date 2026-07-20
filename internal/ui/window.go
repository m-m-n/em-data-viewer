package ui

import (
	"log"
	"strconv"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// minFrameHeight keeps the window grabbable even with zero sections
// (Design: "empty state (zero sections)").
const minFrameHeight = 48

// minContentWidth is deriveContentWidth's positive floor (task0003 Fix 2,
// FR5: "the derived content width must always be positive"): the smallest
// value it ever returns, regardless of how the configured/fallback width
// and chrome combine (including chrome >= width). Any value <= 0 would
// re-enter packRows/packRowsWithBreaks' width-0 degraded case (one section
// per row) at presentation time, which FR5 forbids.
const minContentWidth = 1

// windowIconName is the themed icon name the window declares at
// construction (FR5, NFR2). It is the cross-task contract pinned in
// IMPLEMENTATION.md Shared Components: hicolor theme files, the .desktop
// Icon= value and this runtime name all share this exact string so icon
// resolution succeeds on a system where the deb installed the icons.
// Resolution itself is delegated entirely to the icon theme machinery — on
// a system without the installed icon, gtk.Window.SetIconName is a
// harmless no-op.
const windowIconName = "em-data-viewer"

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

	// settingsWidth is the configured window.width (0 until the first
	// applySettings). refit re-asserts it on every resize instead of
	// reading the width back from the window: on Wayland the map-time
	// configure overwrites the default size with the content's natural
	// size, so a width read back from GTK can silently lose the
	// configured value.
	settingsWidth int

	// mapRefitAttempts counts mapReflowAndRefit's self-re-arms so far
	// (task0003 Fix 4): reset implicitly by never being incremented once
	// shouldRetryMapRefit reports "stop" for either reason (allocation
	// settled, or the retry budget is exhausted).
	mapRefitAttempts int

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
	win.SetIconName(windowIconName)

	root := gtk.NewBox(gtk.OrientationVertical, 0)
	win.SetChild(newContentClip(root))
	blockScrollGestures(win)

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
	attachMonitorRecapTrigger(win, f)

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
		glib.IdleAdd(f.mapReflowAndRefit)
	})
}

// newContentClip hosts content in a structure whose own minimum height does
// not include content's minimum height, so refit's capped SetDefaultSize
// can actually take effect instead of being overridden back up to content's
// full minimum (Design: "Clipping requirement"). A GtkScrolledWindow does
// this only when its vertical scrollbar policy is not NEVER: with any other
// policy its measure() reports a minimum independent of the child (GTK
// falls back to min-content-height, 0 by default) instead of forwarding the
// child's; NEVER is the one policy value that forwards it, reproducing the
// original bug. AUTOMATIC is used here for that reason, not because a
// scrollbar should ever show — both scrollbars are hidden outright below,
// and blockScrollGestures (wired by the caller) stops wheel/touch input
// from moving the hidden overflow into view regardless of policy (AC-2).
// The horizontal policy stays NEVER with natural width propagated, which
// keeps width sized straight through from content exactly as when content
// was the window's direct child (task plan "Preserved behavior": width is
// untouched). refit() measures content directly rather than through this
// host, whose minimum/natural are now deliberately decoupled from it.
func newContentClip(content gtk.Widgetter) *gtk.ScrolledWindow {
	clip := gtk.NewScrolledWindow()
	clip.SetChild(content)
	clip.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	clip.SetPropagateNaturalWidth(true)
	clip.SetPropagateNaturalHeight(false)
	gtk.BaseWidget(clip.HScrollbar()).SetVisible(false)
	gtk.BaseWidget(clip.VScrollbar()).SetVisible(false)
	return clip
}

// blockScrollGestures neutralizes the content clip host's built-in
// scroll-wheel/touchpad handling by claiming every scroll event in the
// CAPTURE phase at the window level, before it can ever reach the clip
// host's own BUBBLE-phase scroll controller (the one that would otherwise
// pan the clipped overflow into view). GTK's event dispatch runs the full
// capture pass root-to-target first and only runs the bubble pass
// afterwards if nothing claimed the event during capture, so claiming here
// skips the bubble pass — and the built-in handler with it — entirely.
// This is the same CAPTURE-phase precedence attachDragToMove below relies
// on. No scrollbar is ever shown (newContentClip) and no gesture can reveal
// hidden content (AC-2; Design: "Clipping requirement").
func blockScrollGestures(win *gtk.ApplicationWindow) {
	scroll := gtk.NewEventControllerScroll(gtk.EventControllerScrollBothAxes | gtk.EventControllerScrollKinetic)
	scroll.SetPropagationPhase(gtk.PhaseCapture)
	scroll.ConnectScroll(func(_, _ float64) bool {
		return true // claim unconditionally; see doc comment above.
	})
	gtk.BaseWidget(win).AddController(scroll)
}

// attachMonitorRecapTrigger observes the window's monitor once realized,
// via its GdkSurface's enter-monitor/leave-monitor signals, and schedules
// the existing deferred reflow+refit whenever it fires, so moving the
// window onto a monitor with a different height re-applies the height cap
// (Design: "Monitor-change re-cap"; AC-3). Registered from the realize
// signal because the surface these signals live on does not exist before
// then.
func attachMonitorRecapTrigger(win *gtk.ApplicationWindow, f *frame) {
	win.ConnectRealize(func() {
		surface := f.toplevelSurface()
		if surface == nil {
			return
		}
		recap := func(*gdk.Monitor) { glib.IdleAdd(f.reflowAndRefit) }
		base := gdk.BaseSurface(surface)
		base.ConnectEnterMonitor(recap)
		base.ConnectLeaveMonitor(recap)
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
		f.settingsWidth = settings.Width
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

// maxMapRefitRetries bounds mapReflowAndRefit's self-re-arming (task0003
// Fix 4, finding 3b93f2acf0fd701d): in a degenerate geometry where the
// content root's allocation never settles positive, retrying without bound
// would re-schedule a glib.IdleAdd callback on every main-loop iteration
// indefinitely, busy-spinning the loop. After this many retries the safety
// net gives up silently instead -- presentation already applied the
// pre-map fit (present/preMapFit), so this path exists only to correct a
// late-settling allocation, not to guarantee one.
const maxMapRefitRetries = 30

// mapReflowAndRefit is the map signal's deferred trigger (task0002 Design:
// "Steady-state safety net"). task0002 presents the window already fitted
// (present/preMapFit below), so this path is reached only as a safety net,
// for the case where the toolkit's first real allocation lands after the
// idle callback this is scheduled from (attachReflowTriggers) runs. It
// re-arms itself for a later main-loop iteration while no allocation
// exists yet, instead of applying (and then permanently no-oping, the way
// running reflowAndRefit directly here once did) a fit computed from a
// width that does not exist (AC-5); once an allocation exists it fits
// exactly once via reflowAndRefit, same as before task0002. The re-arm is
// bounded (task0003 Fix 4, AC-5): once shouldRetryMapRefit reports "stop"
// because the retry budget is exhausted rather than because an allocation
// settled, this gives up silently instead of fitting against a still-zero
// width.
func (f *frame) mapReflowAndRefit() {
	allocatedWidth := f.root.AllocatedWidth()
	if shouldRetryMapRefit(allocatedWidth, f.mapRefitAttempts) {
		f.mapRefitAttempts++
		glib.IdleAdd(f.mapReflowAndRefit)
		return
	}
	if allocatedWidth <= 0 {
		// Retry budget exhausted while still degenerate: give up silently
		// (task0003 Fix 4) rather than fitting against the width-0
		// degraded case refit itself already guards against.
		return
	}
	f.reflowAndRefit()
}

// shouldRetryMapRefit is mapReflowAndRefit's pure retry decision: true
// while the content root has no allocation yet AND the retry budget
// (maxMapRefitRetries) is not yet exhausted, meaning a fit computed now
// would be the width-0 degraded case refit itself already guards against
// via its own AllocatedWidth check. Reports "stop" (false) once either the
// allocation has settled or attempt has reached the bound (task0003 Fix
// 4, AC-5). Pure and GUI-free so the decision is unit-testable without GTK
// (task0002 AC-5, task0003 Test Notes).
func shouldRetryMapRefit(allocatedWidth, attempt int) bool {
	return allocatedWidth <= 0 && attempt < maxMapRefitRetries
}

// present runs the pre-map fit (preMapFit) and shows the window (FR1, FR5,
// US1): the window maps already sized from the real row-packed layout at
// the configured width, so no visible resize follows presentation. Called
// at most once, by UI.Present's first-call-wins guard (IMPLEMENTATION.md
// Shared Components: PresenterUI contract).
func (f *frame) present() {
	f.preMapFit()
	f.win.Present()
}

// preMapFit sizes the window before it is ever mapped, from a width
// derived from the configured width rather than from an allocation
// (task0002 Design "Pre-map fit"; IMPLEMENTATION.md decision 3): GTK
// allocates widgets only after a window is mapped, so a hidden window can
// never satisfy "wait for a settled allocation" the way refit does. The
// derived content width feeds the same explicit-width row-packing seam
// (sectionsView.reflowAt) and the same height formula (fitHeight) that the
// post-map, allocation-driven refit uses, so the two computations agree
// given the same inputs (AC-3). Monitor geometry is unknown before
// realize, so monitorHeight() returns 0 and no cap applies here, exactly
// like refit before the window is realized.
func (f *frame) preMapFit() {
	chrome := f.windowChromeSize()
	windowWidth, _ := f.win.DefaultSize()
	contentWidth := deriveContentWidth(f.settingsWidth, windowWidth, chrome)

	f.sections.reflowAt(contentWidth)
	_, natural, _, _ := f.root.Measure(gtk.OrientationVertical, contentWidth)
	target := natural + chrome
	height := fitHeight(target, minFrameHeight, f.monitorHeight())

	width := f.settingsWidth
	if width <= 0 {
		width = windowWidth
	}
	f.win.SetDefaultSize(width, height)
}

// deriveContentWidth computes the content-area width to row-pack sections
// against before the window has ever been mapped (FR5, IMPLEMENTATION.md
// decision 3): the configured window width minus chrome, or -- before any
// configured width exists (configuredWidth <= 0) -- windowWidth (the
// window's current default width) minus chrome. Never derived from an
// allocation, which does not exist yet pre-map -- reading one would yield
// the width-0 degraded packing this function exists to avoid (AC-4). The
// result is clamped to minContentWidth (task0003 Fix 2, AC-3): when chrome
// is as large as or larger than the configured/fallback width, the raw
// subtraction would be non-positive and re-enter that same forbidden
// degraded packing at presentation time. Pure and GUI-free (task plan Test
// Notes).
func deriveContentWidth(configuredWidth, windowWidth, chrome int) int {
	base := configuredWidth
	if base <= 0 {
		base = windowWidth
	}
	width := base - chrome
	if width < minContentWidth {
		width = minContentWidth
	}
	return width
}

// refit resizes the window's height to fit the root box's current natural
// content height (banner + sections) plus the window's own CSS chrome
// (windowChromeSize), capped at the monitor height once the window is
// realized (task0001 Design: "Shrink requirement", "Monitor cap"). The
// width is re-asserted from the configured value (settingsWidth) at the
// same time. The minimum frame height (NFR2) is additionally enforced
// as a hard floor by win.SetSizeRequest in newFrame; clamping to it here
// too keeps the computed default size consistent with that floor.
//
// Before task0002, omitting the chrome height here was invisible: the
// undersized SetDefaultSize request was always overridden back up by the
// toolkit forcing the window to content's real minimum (the very
// propagation task0002 removes so the monitor cap can hold). With that
// override gone, refit's own target must be accurate on its own.
func (f *frame) refit() {
	contentWidth := f.root.AllocatedWidth()
	if contentWidth <= 0 {
		// Pre-map there is no allocation yet: measuring height-for-width at
		// 0 makes GTK warn and yields nothing usable. The map signal's
		// deferred mapReflowAndRefit re-arms until this re-runs against a
		// real width (AC-5); presentation itself is already fitted via
		// preMapFit, independently of this early-return.
		return
	}
	_, natural, _, _ := f.root.Measure(gtk.OrientationVertical, contentWidth)
	target := natural + f.windowChromeSize()
	height := fitHeight(target, minFrameHeight, f.monitorHeight())

	// Re-assert the configured width rather than reading it back from the
	// window (see settingsWidth). Before the first applySettings there is
	// no configured width yet; keep whatever the window currently has.
	width := f.settingsWidth
	if width <= 0 {
		width, _ = f.win.DefaultSize()
	}
	f.win.SetDefaultSize(width, height)
}

// windowChromeSize is the space the frame's own .dv-window CSS box
// (buildStylesheet/windowCSS) adds around its child on a single axis:
// padding (the fixed windowPadding token) plus the border (the
// user-configurable frame-width style value), each counted on both edges
// of that axis. Padding and border apply symmetrically to all four edges
// (windowCSS's "padding: Npx" / "border: Npx solid ..."), so this same
// value is the chrome contribution on the vertical axis (refit's height
// target) and on the horizontal axis alike (preMapFit's content-width
// derivation, task0002 FR5). This box sits on f.win itself, outside
// f.root — outside the content refit/preMapFit measure directly — so it
// is never included in that measurement and must be added/subtracted
// explicitly.
func (f *frame) windowChromeSize() int {
	return 2*windowPadding + 2*parsePixelLength(f.windowStyle["frame-width"])
}

// parsePixelLength reads a CSS length as pixels, following this app's
// existing "a bare number means pixels" convention (style.go's
// normalizeLength, AC-3 elsewhere): a trailing "px" suffix is stripped
// before parsing. Any other unit — or a value that fails to parse at all —
// contributes 0 rather than erroring: style values are user input, and a
// window resize computation must never fail because of one.
func parsePixelLength(v string) int {
	n, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(v), "px"), 64)
	if err != nil {
		return 0
	}
	return int(n)
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
