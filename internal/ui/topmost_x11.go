package ui

// #cgo pkg-config: gtk4-x11 gtk4
// #cgo CFLAGS: -Wno-deprecated-declarations
// #include <gdk/x11/gdkx.h>
import "C"

import (
	"fmt"
	"log"
	"sync"
	"unsafe"

	coreglib "github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

// This file is the ONLY place in ui (and the whole codebase) that touches
// X11 specifics: the GdkSurface->XID accessor (via a minimal cgo bridge —
// gotk4's generated gdkx11 package does not expose gdk_x11_surface_get_xid)
// and the direct X connection used to send the EWMH client message
// (SPEC/IMPLEMENTATION.md: "X11-specific code lives ONLY in internal/ui,
// never in logic packages").

// gTypeX11Display is the GType of GdkX11Display, used to detect the X11
// (including XWayland) backend at runtime (SPEC FR4).
var gTypeX11Display = coreglib.Type(C.gdk_x11_display_get_type())

// netWMState EWMH _NET_WM_STATE client message actions (EWMH spec section
// "_NET_WM_STATE").
const (
	netWMStateRemove uint32 = 0
	netWMStateAdd    uint32 = 1
	// netWMSourceApplication marks the request as coming from a normal
	// application (EWMH "source indication" data.l[3]).
	netWMSourceApplication uint32 = 1
)

// isX11Display reports whether display's concrete GDK backend is X11
// (including XWayland) — the only backend EWMH keep-above can target (SPEC
// FR4: "Backend detection at runtime from the GDK display type").
func isX11Display(display *gdk.Display) bool {
	if display == nil {
		return false
	}
	return display.IsA(gTypeX11Display)
}

// surfaceXID resolves the X11 window ID (XID) of surface's toplevel from
// its native GdkSurface pointer. ok is false when surface has no native
// pointer yet (not realized) or isn't backed by an X11 surface.
func surfaceXID(surface gdk.Surfacer) (xid uint32, ok bool) {
	if surface == nil {
		return 0, false
	}
	native := gdk.BaseSurface(surface).Native()
	if native == 0 {
		return 0, false
	}
	return uint32(C.gdk_x11_surface_get_xid((*C.GdkSurface)(unsafe.Pointer(native)))), true
}

// ewmhStateChangeData builds the data.l[] payload of a _NET_WM_STATE client
// message that adds or removes a single window-state atom (EWMH spec:
// l[0]=action, l[1]=first property, l[2]=second property (unused here),
// l[3]=source indication, l[4]=unused). Built as a pure function, with no X
// connection involved, so it can be unit-tested (test/README.md; task plan
// Test Notes: "EWMH message construction may be unit-tested if built as a
// pure payload builder").
func ewmhStateChangeData(add bool, stateAtom xproto.Atom) [5]uint32 {
	action := netWMStateRemove
	if add {
		action = netWMStateAdd
	}
	return [5]uint32{action, uint32(stateAtom), 0, netWMSourceApplication, 0}
}

// x11KeepAboveBackend applies _NET_WM_STATE_ABOVE via a direct X connection
// (SPEC: "send the _NET_WM_STATE client message ... over a direct X
// connection") — GTK4 exposes no keep-above API. The connection and the
// interned atoms are resolved lazily on first use and reused; a failure to
// connect permanently degrades this backend to unsupported without ever
// retrying (Risk Assessment: "xgb connection failure on X11 -> degrade to
// unsupported (item insensitive), log").
type x11KeepAboveBackend struct {
	mu            sync.Mutex
	conn          *xgb.Conn
	connectFailed bool

	atomsResolved   bool
	netWMState      xproto.Atom
	netWMStateAbove xproto.Atom
}

func newX11KeepAboveBackend() *x11KeepAboveBackend {
	return &x11KeepAboveBackend{}
}

// Supported reports whether surface has a reachable XID and the X
// connection is (or can be) established (IMPLEMENTATION.md:
// "KeepAboveSupported() bool (true iff X11 backend and XID reachable)").
func (b *x11KeepAboveBackend) Supported(surface gdk.Surfacer) bool {
	if _, ok := surfaceXID(surface); !ok {
		return false
	}
	_, ok := b.connect()
	return ok
}

// Apply adds (above=true) or removes (above=false) _NET_WM_STATE_ABOVE on
// surface's toplevel by sending a _NET_WM_STATE client message to the root
// window (SPEC FR4, AC-3).
func (b *x11KeepAboveBackend) Apply(surface gdk.Surfacer, above bool) error {
	xid, ok := surfaceXID(surface)
	if !ok {
		return fmt.Errorf("ui: keep-above: toplevel has no XID (not realized?)")
	}
	conn, ok := b.connect()
	if !ok {
		return fmt.Errorf("ui: keep-above: no X connection available")
	}
	stateAtom, aboveAtom, err := b.atoms(conn)
	if err != nil {
		return fmt.Errorf("ui: keep-above: resolving EWMH atoms: %w", err)
	}

	data := ewmhStateChangeData(above, aboveAtom)
	event := xproto.ClientMessageEvent{
		Format: 32,
		Window: xproto.Window(xid),
		Type:   stateAtom,
		Data:   xproto.ClientMessageDataUnionData32New(data[:]),
	}
	root := xproto.Setup(conn).DefaultScreen(conn).Root
	mask := uint32(xproto.EventMaskSubstructureNotify | xproto.EventMaskSubstructureRedirect)
	if err := xproto.SendEventChecked(conn, false, root, mask, string(event.Bytes())).Check(); err != nil {
		return fmt.Errorf("ui: keep-above: sending _NET_WM_STATE: %w", err)
	}
	return nil
}

// connect lazily opens (and remembers) the X connection used for all EWMH
// requests. Once a connection attempt fails, it is never retried (the
// process's X11-ness doesn't change at runtime).
func (b *x11KeepAboveBackend) connect() (*xgb.Conn, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn != nil {
		return b.conn, true
	}
	if b.connectFailed {
		return nil, false
	}
	conn, err := xgb.NewConn()
	if err != nil {
		b.connectFailed = true
		log.Printf("ui: keep-above: opening X connection failed, degrading to unsupported: %v", err)
		return nil, false
	}
	b.conn = conn
	return conn, true
}

// atoms resolves and caches the "_NET_WM_STATE" and "_NET_WM_STATE_ABOVE"
// atoms.
func (b *x11KeepAboveBackend) atoms(conn *xgb.Conn) (state, above xproto.Atom, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.atomsResolved {
		return b.netWMState, b.netWMStateAbove, nil
	}

	stateName := "_NET_WM_STATE"
	stateReply, err := xproto.InternAtom(conn, false, uint16(len(stateName)), stateName).Reply()
	if err != nil {
		return 0, 0, err
	}
	aboveName := "_NET_WM_STATE_ABOVE"
	aboveReply, err := xproto.InternAtom(conn, false, uint16(len(aboveName)), aboveName).Reply()
	if err != nil {
		return 0, 0, err
	}

	b.netWMState = stateReply.Atom
	b.netWMStateAbove = aboveReply.Atom
	b.atomsResolved = true
	return b.netWMState, b.netWMStateAbove, nil
}
