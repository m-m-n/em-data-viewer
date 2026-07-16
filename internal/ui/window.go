package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// minFrameHeight keeps the window grabbable even with zero sections
// (Design: "empty state (zero sections)").
const minFrameHeight = 48

// frame is the window chrome: an undecorated GtkApplicationWindow holding a
// vertical stack of the config-error banner and the sections view, with a
// live CSS provider, drag-to-move and the right-click context menu.
type frame struct {
	win      *gtk.ApplicationWindow
	root     *gtk.Box
	banner   *banner
	sections *sectionsView
	css      *gtk.CSSProvider

	windowStyle map[string]string
}

// newFrame builds the undecorated frame window (FR1) and wires
// drag-to-move (AC-3) and the right-click context menu (AC-7). quit is
// invoked by the context menu's 終了 item.
func newFrame(app *gtk.Application, quit func()) *frame {
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

	f := &frame{
		win:         win,
		root:        root,
		banner:      b,
		sections:    sv,
		css:         gtk.NewCSSProvider(),
		windowStyle: MergeStyle(WindowStyleDefaults, nil),
	}

	if display := gdk.DisplayGetDefault(); display != nil {
		gtk.StyleContextAddProviderForDisplay(display, f.css, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	}
	f.restyle()

	attachDragToMove(win)
	attachContextMenu(win, quit)

	return f
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

// attachDragToMove starts a compositor-side interactive move whenever the
// primary button is pressed on widget's background (FR1, AC-3;
// Wayland-compatible — never uses absolute positioning).
func attachDragToMove(widget gtk.Widgetter) {
	root := gtk.BaseWidget(widget)

	drag := gtk.NewGestureClick()
	drag.SetButton(1) // primary (left) button only
	drag.ConnectPressed(func(_ int, x, y float64) {
		native := root.Native()
		if native == nil {
			return
		}
		toplevel, ok := native.Surface().(gdk.Topleveller)
		if !ok {
			return
		}
		nx, ny, ok := root.TranslateCoordinates(native, x, y)
		if !ok {
			return
		}
		device := drag.Device()
		if device == nil {
			return
		}
		var timestamp uint32
		if event := drag.CurrentEvent(); event != nil {
			timestamp = gdk.BaseEvent(event).Time()
		}
		toplevel.BeginMove(device, int(drag.CurrentButton()), nx, ny, timestamp)
	})
	root.AddController(drag)
}
