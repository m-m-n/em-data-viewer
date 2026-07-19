package ui

import "testing"

// TestFitHeight_ClampsToMinimumAndMonitorCap references task0001 AC-7/AC-8:
// the target height is never below minHeight (NFR2, the empty-state grab
// handle), and never above a known monitor height (FR3); before the window
// is realized (monitorHeight == 0), no cap applies. Kept as a pure,
// GUI-free helper per the task plan's test notes so the sizing arithmetic
// is unit-testable independently of GTK measurement/monitor lookups.
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
// minimum frame height stays 48px (unchanged by task0001).
func TestMinFrameHeight_MatchesDesignConstant(t *testing.T) {
	if minFrameHeight != 48 {
		t.Errorf("minFrameHeight = %d, want 48", minFrameHeight)
	}
}
