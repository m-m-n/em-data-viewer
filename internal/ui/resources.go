package ui

import (
	_ "embed"
	"log"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// iconResourceBytes is the compiled GResource bundle produced by the
// Makefile `resources` target from assets/icon.gresource.xml
// (IMPLEMENTATION.md: "Bundle generated inside internal/ui, git-ignored").
// go:embed can only reach files inside this package's own directory tree,
// which is why the bundle is generated here rather than under build/; a
// fresh checkout cannot compile this package until `make resources` (or
// `make build` / `make test`, which depend on it) has run once.
//
//go:embed icon.gresource
var iconResourceBytes []byte

// iconResourceBasePath is the base resource path added to the icon
// theme's search path by registerIconResources below. It must match,
// verbatim, the <gresource prefix=...> in assets/icon.gresource.xml -- the
// cross-task contract pinned in IMPLEMENTATION.md Shared Components
// ("Resource base path").
const iconResourceBasePath = "/net/laser5/em-data-viewer/icons"

// parseIconResource decodes iconResourceBytes into a *gio.Resource. Split
// out from registerIconResources so this GTK-display-independent half is
// covered by a headless test (task0001 Test Notes; SPEC.md Unit Tests).
func parseIconResource() (*gio.Resource, error) {
	return gio.NewResourceFromData(glib.NewBytes(iconResourceBytes))
}

// registerIconResources registers the embedded GResource bundle globally
// and adds iconResourceBasePath to the default display's icon theme
// resource path, so windowIconName (window.go) resolves em-data-viewer
// from the bundle without requiring installed hicolor icons (SPEC FR3).
//
// Precondition: called after GTK/GApplication activation has begun and
// before any window is created (New's ConnectActivate callback, ui.go).
// On any failure -- malformed embedded data, or no default display -- this
// logs one line and returns without panicking; theme-installed icons (if
// present on the system) remain the fallback.
func registerIconResources() {
	resource, err := parseIconResource()
	if err != nil {
		log.Printf("ui: register icon resources: %v", err)
		return
	}
	gio.ResourcesRegister(resource)

	display := gdk.DisplayGetDefault()
	if display == nil {
		log.Printf("ui: register icon resources: no default display")
		return
	}
	gtk.IconThemeGetForDisplay(display).AddResourcePath(iconResourceBasePath)
}
