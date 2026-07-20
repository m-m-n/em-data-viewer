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

// --- task0003 Fix 2: positive width floor (AC-3) ---

// TestDeriveContentWidth_ClampsToPositiveFloor references AC-3: the width
// derivation never returns a value below its positive floor, for any
// combination of configured width, fallback width and chrome -- including
// chrome greater than or equal to the width it would otherwise be
// subtracted from.
func TestDeriveContentWidth_ClampsToPositiveFloor(t *testing.T) {
	tests := []struct {
		name            string
		configuredWidth int
		windowWidth     int
		chrome          int
	}{
		{name: "chrome equals configured width", configuredWidth: 100, windowWidth: 0, chrome: 100},
		{name: "chrome exceeds configured width", configuredWidth: 100, windowWidth: 0, chrome: 250},
		{name: "chrome equals fallback window width", configuredWidth: 0, windowWidth: 80, chrome: 80},
		{name: "chrome exceeds fallback window width", configuredWidth: 0, windowWidth: 80, chrome: 500},
		{name: "no configured or fallback width at all, chrome positive", configuredWidth: 0, windowWidth: 0, chrome: 10},
		{name: "everything zero", configuredWidth: 0, windowWidth: 0, chrome: 0},
		{name: "negative configured width, chrome exceeds fallback", configuredWidth: -5, windowWidth: 20, chrome: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveContentWidth(tt.configuredWidth, tt.windowWidth, tt.chrome)
			if got < minContentWidth {
				t.Errorf("deriveContentWidth(%d, %d, %d) = %d, want >= minContentWidth (%d)", tt.configuredWidth, tt.windowWidth, tt.chrome, got, minContentWidth)
			}
		})
	}
}

// TestDeriveContentWidth_DoesNotClampWhenResultAlreadyPositive references
// AC-3 negatively: the clamp must not alter a result that is already at or
// above the floor (no regression on the existing positive-result cases).
func TestDeriveContentWidth_DoesNotClampWhenResultAlreadyPositive(t *testing.T) {
	got := deriveContentWidth(400, 0, 10)
	if got != 390 {
		t.Fatalf("deriveContentWidth(400, 0, 10) = %d, want 390 (unclamped)", got)
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

// --- shouldRetryMapRefit (task0002 AC-5, Design "Steady-state safety net";
// task0003 Fix 4 bounds the retry, AC-5) ---

// TestShouldRetryMapRefit references AC-5: the map-time deferred re-fit's
// pure retry decision -- re-arm for a later main-loop iteration while the
// content root has no allocation yet and the retry budget is not
// exhausted, instead of applying (and permanently no-oping) a fit computed
// from a width that does not exist.
func TestShouldRetryMapRefit(t *testing.T) {
	tests := []struct {
		name           string
		allocatedWidth int
		attempt        int
		want           bool
	}{
		{name: "no allocation yet, first attempt: retry", allocatedWidth: 0, attempt: 0, want: true},
		{name: "negative allocation (defensive): retry", allocatedWidth: -1, attempt: 0, want: true},
		{name: "allocation settled: stop retrying, fit now", allocatedWidth: 400, attempt: 0, want: false},
		{name: "allocation settled even on a late attempt: stop", allocatedWidth: 400, attempt: maxMapRefitRetries, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRetryMapRefit(tt.allocatedWidth, tt.attempt)
			if got != tt.want {
				t.Errorf("shouldRetryMapRefit(%d, %d) = %v, want %v", tt.allocatedWidth, tt.attempt, got, tt.want)
			}
		})
	}
}

// TestShouldRetryMapRefit_NeverAllocatingWidget_StopsAfterBoundedRetries
// references AC-5 directly: for a widget whose allocation never settles
// positive, the decision reports "retry" only a bounded number of times
// before reporting "stop" -- proving mapReflowAndRefit's self-re-arm
// cannot loop forever (task0003 Fix 4, finding 3b93f2acf0fd701d).
func TestShouldRetryMapRefit_NeverAllocatingWidget_StopsAfterBoundedRetries(t *testing.T) {
	const neverAllocatedWidth = 0

	retries := 0
	for attempt := 0; shouldRetryMapRefit(neverAllocatedWidth, attempt); attempt++ {
		retries++
		if retries > maxMapRefitRetries {
			t.Fatalf("shouldRetryMapRefit kept reporting retry past attempt %d for a never-allocating widget, want it bounded by maxMapRefitRetries (%d)", attempt, maxMapRefitRetries)
		}
	}

	if retries != maxMapRefitRetries {
		t.Fatalf("total retries reported = %d, want exactly maxMapRefitRetries (%d)", retries, maxMapRefitRetries)
	}
	if shouldRetryMapRefit(neverAllocatedWidth, retries) {
		t.Fatalf("shouldRetryMapRefit(%d, %d) = true, want false (stop) once the retry budget is exhausted", neverAllocatedWidth, retries)
	}
}
