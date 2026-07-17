package ui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// keepAboveMenuLabel is the base label of the「最前面に固定」check item
// (SPEC US2, FR3).
const keepAboveMenuLabel = "最前面に固定"

// keepAboveMenu is the live handle the owner (frame) uses to push
// keep-above state changes into the already-built popover menu (AC-5): the
// bound GAction's enabled/state properties drive the rendered item
// automatically, so no menu rebuild is needed for check-mark/sensitivity
// changes.
type keepAboveMenu struct {
	action *gio.SimpleAction
}

// setState applies state to the check item: sensitivity from
// state.Sensitive and the check mark from state.Checked (AC-1, AC-4, AC-5).
func (m *keepAboveMenu) setState(state keepAboveState) {
	if m == nil || m.action == nil {
		return
	}
	m.action.SetEnabled(state.Sensitive)
	m.action.SetState(glib.NewVariantBoolean(state.Checked))
}

// attachContextMenu wires a right-click (secondary button) gesture on
// widget that opens a stock GTK popover menu with the「最前面に固定」check
// item and a 終了 item invoking quit (FR3, FR8, AC-7). initialState seeds
// the check item's label (the Wayland hint, baked in once since the GDK
// backend never changes at runtime) and its starting sensitivity/check
// mark. onToggle is invoked with the NEW desired value whenever the user
// activates the check item; it never fires while the item is insensitive
// (AC-4: no X11 code path executes on Wayland). No custom menu styling
// (SPEC FR4).
func attachContextMenu(widget gtk.Widgetter, quit func(), initialState keepAboveState, onToggle func(enabled bool)) *keepAboveMenu {
	root := gtk.BaseWidget(widget)

	actions := gio.NewSimpleActionGroup()

	quitAction := gio.NewSimpleAction("quit", nil)
	quitAction.ConnectActivate(func(_ *glib.Variant) {
		if quit != nil {
			quit()
		}
	})
	actions.Insert(quitAction)

	// A stateful boolean action with no handler on "activate": per GIO,
	// clicking it forwards to "change-state" with the flipped value by
	// default. We only react on "change-state" and deliberately do not
	// call SetState there ourselves — the check mark only flips once the
	// owner confirms the new state via keepAboveMenu.setState, so it never
	// shows a value that wasn't actually applied.
	keepAboveAction := gio.NewSimpleActionStateful("keep_above", nil, glib.NewVariantBoolean(initialState.Checked))
	keepAboveAction.SetEnabled(initialState.Sensitive)
	keepAboveAction.ConnectChangeState(func(value *glib.Variant) {
		if onToggle != nil {
			onToggle(value.Boolean())
		}
	})
	actions.Insert(keepAboveAction)

	root.InsertActionGroup("dv-menu", actions)

	label := keepAboveMenuLabel
	if initialState.Hint != "" {
		label = keepAboveMenuLabel + "（" + initialState.Hint + "）"
	}

	model := gio.NewMenu()
	model.Append(label, "dv-menu.keep_above")
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

	return &keepAboveMenu{action: keepAboveAction}
}
