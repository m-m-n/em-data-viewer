package ui

import "testing"

// TestIconResourceBytes_NonEmpty references task0001 AC-7's pure-logic
// slice: registerIconResources itself needs a real GTK display and is
// excluded from headless tests (test/README.md), but the embedded bundle
// it reads from must at least contain data -- this catches a `make
// resources` run that produced an empty or missing file (the generated
// bundle is git-ignored; go:embed reads whatever `make resources` most
// recently produced).
func TestIconResourceBytes_NonEmpty(t *testing.T) {
	if len(iconResourceBytes) == 0 {
		t.Fatal("iconResourceBytes is empty; run `make resources` before `go test`")
	}
}

// TestIconResourceBytes_ParsesAsResource references SPEC.md's Unit Tests
// scenario ("the embedded resource data is non-empty and registers without
// error"): parsing does not require a GTK display (unlike registration's
// icon-theme half), so it is safe to run headlessly and catches a
// corrupted/truncated bundle without needing GTK.
func TestIconResourceBytes_ParsesAsResource(t *testing.T) {
	resource, err := parseIconResource()
	if err != nil {
		t.Fatalf("parseIconResource() error = %v, want nil", err)
	}
	if resource == nil {
		t.Fatal("parseIconResource() = nil resource, want non-nil")
	}
}

// TestIconResourceBasePath_MatchesManifestPrefix references
// IMPLEMENTATION.md Shared Components' "Resource base path" contract: the
// runtime registration and the gresource manifest's <gresource prefix=...>
// must agree verbatim.
func TestIconResourceBasePath_MatchesManifestPrefix(t *testing.T) {
	want := "/net/laser5/em-data-viewer/icons"
	if iconResourceBasePath != want {
		t.Errorf("iconResourceBasePath = %q, want %q", iconResourceBasePath, want)
	}
}
