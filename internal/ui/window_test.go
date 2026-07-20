package ui

import "testing"

// TestFitHeight_ClampsToMinimumAndMonitorCap references task0001 AC-7/AC-8
// and task0002 AC-1: the target height is never below minHeight (NFR2, the
// empty-state grab handle), and never above a known monitor height (FR3);
// before the window is realized (monitorHeight == 0), no cap applies. This
// formula is unchanged by task0002 — that task's content-clip host
// restructuring (window.go) is what makes the toolkit actually allocate
// the window at this computed height instead of forcing it back up to
// content's minimum. Kept as a pure, GUI-free helper per the task plan's
// test notes so the sizing arithmetic is unit-testable independently of
// GTK measurement/monitor lookups.
func TestFitHeight_ClampsToMinimumAndMonitorCap(t *testing.T) {
	tests := []struct {
		name           string
		contentNatural int
		minHeight      int
		monitorHeight  int
		want           int
	}{
		{
			name:           "content within bounds is used as-is",
			contentNatural: 300,
			minHeight:      48,
			monitorHeight:  1080,
			want:           300,
		},
		{
			name:           "content shorter than minimum is clamped up",
			contentNatural: 10,
			minHeight:      48,
			monitorHeight:  1080,
			want:           48,
		},
		{
			name:           "content taller than the monitor is clamped down",
			contentNatural: 2000,
			minHeight:      48,
			monitorHeight:  1080,
			want:           1080,
		},
		{
			name:           "zero sections still clamps to the minimum",
			contentNatural: 0,
			minHeight:      48,
			monitorHeight:  1080,
			want:           48,
		},
		{
			name:           "no monitor known yet applies no cap",
			contentNatural: 5000,
			minHeight:      48,
			monitorHeight:  0,
			want:           5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fitHeight(tt.contentNatural, tt.minHeight, tt.monitorHeight)
			if got != tt.want {
				t.Errorf("fitHeight(%d, %d, %d) = %d, want %d", tt.contentNatural, tt.minHeight, tt.monitorHeight, got, tt.want)
			}
		})
	}
}

// TestMinFrameHeight_MatchesDesignConstant references AC-8: the empty-state
// minimum frame height stays 48px (unchanged by task0001 and task0002's
// clipping-structure restructuring — task0002 Preserved behavior).
func TestMinFrameHeight_MatchesDesignConstant(t *testing.T) {
	if minFrameHeight != 48 {
		t.Errorf("minFrameHeight = %d, want 48", minFrameHeight)
	}
}

// TestWindowIconName_MatchesDesignConstant references app-icon task0002
// AC-1: the window icon name constant is exactly "em-data-viewer" — the
// cross-task contract pinned in IMPLEMENTATION.md Shared Components.
// Wiring it into gtk.Window.SetIconName at window construction (newFrame)
// requires a real GTK window/display and is verified manually (task plan
// Test Notes; test/README.md: GTK/GUI layers are excluded from automated
// tests).
func TestWindowIconName_MatchesDesignConstant(t *testing.T) {
	if windowIconName != "em-data-viewer" {
		t.Errorf("windowIconName = %q, want %q", windowIconName, "em-data-viewer")
	}
}

// TestParsePixelLength_ReadsBareNumbersAsPixels references task0002 AC-4:
// refit's chrome-height addition (windowChromeSize) depends on reading
// the user-configurable frame-width style value as pixels the same way the
// rest of this app treats a bare number as pixels (style.go's
// normalizeLength). New pure helper introduced by task0002 (Test Notes),
// so it is covered directly rather than only through GTK-level rendering.
func TestParsePixelLength_ReadsBareNumbersAsPixels(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{name: "plain pixel value", in: "1px", want: 1},
		{name: "multi-digit pixel value", in: "12px", want: 12},
		{name: "fractional pixel value truncates", in: "2.5px", want: 2},
		{name: "surrounding whitespace is trimmed", in: "  3px  ", want: 3},
		{name: "zero", in: "0px", want: 0},
		{name: "empty value contributes nothing", in: "", want: 0},
		{name: "non-numeric value contributes nothing", in: "thick", want: 0},
		{name: "non-pixel unit contributes nothing", in: "1em", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePixelLength(tt.in)
			if got != tt.want {
				t.Errorf("parsePixelLength(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
