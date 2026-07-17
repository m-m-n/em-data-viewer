package ui

import (
	"testing"

	"github.com/jezek/xgb/xproto"
)

// TestEwmhStateChangeData builds the _NET_WM_STATE client message payload
// as a pure function, independent of any live X connection (task plan Test
// Notes: "EWMH message construction may be unit-tested if built as a pure
// payload builder").
func TestEwmhStateChangeData(t *testing.T) {
	const aboveAtom xproto.Atom = 123

	add := ewmhStateChangeData(true, aboveAtom)
	wantAdd := [5]uint32{netWMStateAdd, uint32(aboveAtom), 0, netWMSourceApplication, 0}
	if add != wantAdd {
		t.Errorf("ewmhStateChangeData(true, %d) = %v, want %v", aboveAtom, add, wantAdd)
	}

	remove := ewmhStateChangeData(false, aboveAtom)
	wantRemove := [5]uint32{netWMStateRemove, uint32(aboveAtom), 0, netWMSourceApplication, 0}
	if remove != wantRemove {
		t.Errorf("ewmhStateChangeData(false, %d) = %v, want %v", aboveAtom, remove, wantRemove)
	}
}

// TestIsX11Display_NilIsFalse guards the nil-display path so a display
// that can't be classified never reports X11 support.
func TestIsX11Display_NilIsFalse(t *testing.T) {
	if isX11Display(nil) {
		t.Error("isX11Display(nil) = true, want false")
	}
}

// TestSurfaceXID_NilIsUnreachable guards the not-realized/nil-surface path.
func TestSurfaceXID_NilIsUnreachable(t *testing.T) {
	if _, ok := surfaceXID(nil); ok {
		t.Error("surfaceXID(nil) ok = true, want false")
	}
}
