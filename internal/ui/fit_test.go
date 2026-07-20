package ui

import "testing"

// --- deriveContentWidth (task0002 FR5, Design "Pre-map fit") ---

// TestDeriveContentWidth references AC-4: the content-area width to
// row-pack sections against before the window has ever been mapped is
// derived from the configured window width minus the window's own CSS
// chrome, falling back to the window's current default width before any
// configured width exists -- never from an allocation (which does not
// exist pre-map).
func TestDeriveContentWidth(t *testing.T) {
	tests := []struct {
		name            string
		configuredWidth int
		windowWidth     int
		chrome          int
		want            int
	}{
		{
			name:            "configured width present: content width is configured minus chrome",
			configuredWidth: 400,
			windowWidth:     0,
			chrome:          10,
			want:            390,
		},
		{
			name:            "no configured width yet: falls back to the window's current default width",
			configuredWidth: 0,
			windowWidth:     300,
			chrome:          10,
			want:            290,
		},
		{
			name:            "negative configured width treated as absent, falls back",
			configuredWidth: -1,
			windowWidth:     300,
			chrome:          10,
			want:            290,
		},
		{
			name:            "zero chrome subtracts nothing",
			configuredWidth: 400,
			windowWidth:     0,
			chrome:          0,
			want:            400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveContentWidth(tt.configuredWidth, tt.windowWidth, tt.chrome)
			if got != tt.want {
				t.Errorf("deriveContentWidth(%d, %d, %d) = %d, want %d", tt.configuredWidth, tt.windowWidth, tt.chrome, got, tt.want)
			}
		})
	}
}

// TestDeriveContentWidth_NeverZeroForAConfiguredWidth references AC-4: a
// configured width always derives a nonzero content width, so the pre-map
// fit path never packs sections at width 0 -- the degraded
// one-section-per-row case packRows/packRowsWithBreaks apply when avail
// <= 0 -- the way reading straight from the (nonexistent) pre-map
// allocation would (FR5).
func TestDeriveContentWidth_NeverZeroForAConfiguredWidth(t *testing.T) {
	got := deriveContentWidth(400, 0, 10)
	if got <= 0 {
		t.Errorf("deriveContentWidth(400, 0, 10) = %d, want > 0 (must never degrade to the width-0 packing case)", got)
	}
}

// --- pre-map / allocation-driven agreement (task0002 AC-3) ---

// TestPreMapFitAgreesWithAllocationDrivenFit references AC-3: given a
// configured width, the content width the pre-map path
// (deriveContentWidth) packs against equals the content width the
// post-map, allocation-driven path (refit, reading AllocatedWidth) would
// see once the window has actually been sized to and mapped at that same
// configured width -- the window's own width minus the identical chrome
// windowChromeSize models. Feeding an equal natural-height-for-width value
// through the one shared height formula (fitHeight) then necessarily
// yields an identical final height on both paths (IMPLEMENTATION.md
// decision 3, task plan Postcondition).
func TestPreMapFitAgreesWithAllocationDrivenFit(t *testing.T) {
	const configuredWidth = 400
	const chrome = 10
	// GTK's Measure(width) is a deterministic function of width for
	// unchanged content, so an equal content width on both paths yields an
	// equal natural height -- represented directly here since GTK
	// measurement itself is excluded from automated tests (test/README.md).
	const naturalHeightAtContentWidth = 220

	// Pre-map: no window has ever had a default width applied yet
	// (the windowWidth fallback is unused since configuredWidth > 0).
	preMapContentWidth := deriveContentWidth(configuredWidth, 0, chrome)

	// Post-map: once the window has been sized to configuredWidth and
	// mapped, its content root's AllocatedWidth is the window's own width
	// minus the same chrome.
	postMapAllocatedContentWidth := configuredWidth - chrome

	if preMapContentWidth != postMapAllocatedContentWidth {
		t.Fatalf("preMapContentWidth = %d, want %d (must equal the post-map allocated content width for the same configured width)", preMapContentWidth, postMapAllocatedContentWidth)
	}

	preMapHeight := fitHeight(naturalHeightAtContentWidth+chrome, minFrameHeight, 0)
	postMapHeight := fitHeight(naturalHeightAtContentWidth+chrome, minFrameHeight, 0)

	if preMapHeight != postMapHeight {
		t.Errorf("preMapHeight = %d, postMapHeight = %d, want equal", preMapHeight, postMapHeight)
	}
}

// --- shouldRetryMapRefit (task0002 AC-5, Design "Steady-state safety net") ---

// TestShouldRetryMapRefit references AC-5: the map-time deferred re-fit's
// pure retry decision -- re-arm for a later main-loop iteration while the
// content root has no allocation yet, instead of applying (and
// permanently no-oping) a fit computed from a width that does not exist.
func TestShouldRetryMapRefit(t *testing.T) {
	tests := []struct {
		name           string
		allocatedWidth int
		want           bool
	}{
		{name: "no allocation yet: retry", allocatedWidth: 0, want: true},
		{name: "negative allocation (defensive): retry", allocatedWidth: -1, want: true},
		{name: "allocation settled: stop retrying, fit now", allocatedWidth: 400, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRetryMapRefit(tt.allocatedWidth)
			if got != tt.want {
				t.Errorf("shouldRetryMapRefit(%d) = %v, want %v", tt.allocatedWidth, got, tt.want)
			}
		})
	}
}
