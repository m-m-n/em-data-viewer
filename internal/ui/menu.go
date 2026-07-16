package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// attachContextMenu wires a right-click (secondary button) gesture on
// widget that opens a stock GTK popover menu with a single 終了 item
// invoking quit (FR8, AC-7). No custom menu styling.
func attachContextMenu(widget gtk.Widgetter, quit func()) {
	root := gtk.BaseWidget(widget)

	actions := gio.NewSimpleActionGroup()
	quitAction := gio.NewSimpleAction("quit", nil)
	quitAction.ConnectActivate(func(_ *glib.Variant) {
		if quit != nil {
			quit()
		}
	})
	actions.Insert(quitAction)
	root.InsertActionGroup("dv-menu", actions)

	model := gio.NewMenu()
	model.Append("終了", "dv-menu.quit")

	popover := gtk.NewPopoverMenuFromModel(model)
	popover.SetParent(root)
	popover.SetHasArrow(false)

	click := gtk.NewGestureClick()
	click.SetButton(3) // secondary (right) button only
	click.ConnectPressed(func(_ int, x, y float64) {
		rect := gdk.NewRectangle(int(x), int(y), 1, 1)
		popover.SetPointingTo(&rect)
		popover.Popup()
	})
	root.AddController(click)
}
