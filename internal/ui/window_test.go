package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
)

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

// --- Whole-window opacity toggle (task0002) ---

// TestIsToggleOpacityKeyEvent references AC-6: the pure match rule accepts
// the lowercase or uppercase `t` keyval with the control modifier present
// and neither alt nor super held, and rejects every other combination (no
// control, control+alt, control+super, and an unrelated keyval with control
// held). The match rule itself is unchanged by this task; only the
// identifier is renamed (IMPLEMENTATION.md Conventions "Naming: pattern
// identifiers").
func TestIsToggleOpacityKeyEvent(t *testing.T) {
	tests := []struct {
		name   string
		keyval uint
		state  gdk.ModifierType
		want   bool
	}{
		{name: "lowercase t with control", keyval: gdk.KEY_t, state: gdk.ControlMask, want: true},
		{name: "uppercase T with control (Caps Lock/Shift)", keyval: gdk.KEY_T, state: gdk.ControlMask, want: true},
		{name: "lowercase t with no control", keyval: gdk.KEY_t, state: gdk.NoModifierMask, want: false},
		{name: "uppercase T with no control", keyval: gdk.KEY_T, state: gdk.NoModifierMask, want: false},
		{name: "t with control and alt", keyval: gdk.KEY_t, state: gdk.ControlMask | gdk.AltMask, want: false},
		{name: "t with control and super", keyval: gdk.KEY_t, state: gdk.ControlMask | gdk.SuperMask, want: false},
		{name: "unrelated key with control held", keyval: gdk.KEY_a, state: gdk.ControlMask, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isToggleOpacityKeyEvent(tt.keyval, tt.state); got != tt.want {
				t.Errorf("isToggleOpacityKeyEvent(%v, %v) = %v, want %v", tt.keyval, tt.state, got, tt.want)
			}
		})
	}
}

// TestFrame_OpacityPatternActive_StartsPrimary references AC-1: a freshly
// constructed frame reports the primary pattern as active. Exercised on a
// zero-value frame -- no GTK display required (Test Notes).
func TestFrame_OpacityPatternActive_StartsPrimary(t *testing.T) {
	f := &frame{}

	if f.opacityPatternActive() {
		t.Fatalf("newly constructed frame reports the alternate pattern active, want primary (AC-1)")
	}
}

// TestFrame_FlipOpacityPattern_TogglesWithoutDriftAndDoesNotTouchStyleMap
// references AC-2: the flip helper alternates false -> true -> false across
// successive calls with no accumulated drift, and never modifies the
// merged style map (the config-derived map applySettings produces).
func TestFrame_FlipOpacityPattern_TogglesWithoutDriftAndDoesNotTouchStyleMap(t *testing.T) {
	f := &frame{windowStyle: map[string]string{"opacity": "1.0", "opacity_alt": "0.45"}}
	before := make(map[string]string, len(f.windowStyle))
	for k, v := range f.windowStyle {
		before[k] = v
	}

	wantAfterFlip := []bool{true, false, true, false}
	for i, want := range wantAfterFlip {
		f.flipOpacityPattern()
		if got := f.opacityPatternActive(); got != want {
			t.Errorf("after flip #%d: opacityPatternActive() = %v, want %v (AC-2)", i+1, got, want)
		}
	}

	if len(f.windowStyle) != len(before) {
		t.Fatalf("flipping changed the style map size: got %d keys, want %d (AC-2)", len(f.windowStyle), len(before))
	}
	for k, want := range before {
		if f.windowStyle[k] != want {
			t.Errorf("flipping wrote into the style map[%q] = %q, want unchanged %q (AC-2)", k, f.windowStyle[k], want)
		}
	}
}

// TestFrame_EffectiveOpacity_SelectsPrimaryOrAlternateByActivePattern
// references AC-3: the effective opacity the frame would apply is the
// primary configured value when the primary pattern is active and the
// alternate configured value when the alternate pattern is active, for a
// frame whose style map carries both values. The style map is constructed
// directly rather than through MergeStyle (Test Notes; the opacity keys'
// presence in WindowStyleDefaults is task0001's territory).
func TestFrame_EffectiveOpacity_SelectsPrimaryOrAlternateByActivePattern(t *testing.T) {
	f := &frame{windowStyle: map[string]string{"opacity": "0.3", "opacity_alt": "0.6"}}

	if got := EffectiveOpacity(f.windowStyle, f.opacityPatternActive()); got != 0.3 {
		t.Errorf("EffectiveOpacity() with primary active = %v, want 0.3 (AC-3)", got)
	}

	f.flipOpacityPattern()
	if got := EffectiveOpacity(f.windowStyle, f.opacityPatternActive()); got != 0.6 {
		t.Errorf("EffectiveOpacity() with alternate active = %v, want 0.6 (AC-3)", got)
	}
}

// TestFrame_EffectiveOpacity_SettingsReapplicationKeepsActivePatternAndPicksUpNewValue
// references AC-4, the reload regression: re-applying settings with new
// style values while the alternate pattern is active leaves the alternate
// pattern active and yields the newly configured alternate opacity as the
// effective value. applySettings' own body does nothing more than
// reassign f.windowStyle before applying, so that reassignment is
// simulated directly here (Test Notes) rather than calling the
// GTK-dependent applySettings/applyOpacity methods.
func TestFrame_EffectiveOpacity_SettingsReapplicationKeepsActivePatternAndPicksUpNewValue(t *testing.T) {
	f := &frame{windowStyle: map[string]string{"opacity": "1.0", "opacity_alt": "0.45"}}
	f.flipOpacityPattern() // alternate now active

	f.windowStyle = map[string]string{"opacity": "1.0", "opacity_alt": "0.2"}

	if !f.opacityPatternActive() {
		t.Fatalf("re-applying window settings cleared the active pattern, want it to remain alternate (AC-4)")
	}
	if got := EffectiveOpacity(f.windowStyle, f.opacityPatternActive()); got != 0.2 {
		t.Errorf("EffectiveOpacity() = %v, want the newly configured alternate value 0.2 (AC-4)", got)
	}
}

// TestFrame_Stylesheet_IndependentOfActiveOpacityPattern references AC-5:
// the stylesheet built by the frame is byte-identical for both values of
// the active pattern (the toggle no longer affects CSS), and the window's
// background color in that stylesheet is the configured primary background
// in both cases. Comparing the two stylesheet strings directly also guards
// against a future reintroduction of the CSS-side toggle (Test Notes).
func TestFrame_Stylesheet_IndependentOfActiveOpacityPattern(t *testing.T) {
	f := &frame{windowStyle: MergeStyle(WindowStyleDefaults, nil)}

	wantBackground := fmt.Sprintf("background-color: %s;", f.windowStyle["background"])

	primaryCSS := buildStylesheet(f.windowStyle, nil)
	if !strings.Contains(primaryCSS, wantBackground) {
		t.Fatalf("primary stylesheet = %q, want it to contain %q", primaryCSS, wantBackground)
	}

	f.flipOpacityPattern() // alternate now active
	altCSS := buildStylesheet(f.windowStyle, nil)

	if altCSS != primaryCSS {
		t.Errorf("stylesheet changed when the active pattern flipped, want it byte-identical (AC-5):\nprimary=%q\nalt=%q", primaryCSS, altCSS)
	}
	if !strings.Contains(altCSS, wantBackground) {
		t.Errorf("alternate stylesheet = %q, want it to still contain the primary background %q (AC-5)", altCSS, wantBackground)
	}
}
