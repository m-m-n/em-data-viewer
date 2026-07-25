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

// --- Background pattern toggle (task0002) ---

// TestIsToggleBackgroundKeyEvent references AC-1 and AC-2: the pure match
// rule accepts the lowercase or uppercase `t` keyval with the control
// modifier present and neither alt nor super held, and rejects every other
// combination (no control, control+alt, control+super, and an unrelated
// keyval with control held).
func TestIsToggleBackgroundKeyEvent(t *testing.T) {
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
			if got := isToggleBackgroundKeyEvent(tt.keyval, tt.state); got != tt.want {
				t.Errorf("isToggleBackgroundKeyEvent(%v, %v) = %v, want %v", tt.keyval, tt.state, got, tt.want)
			}
		})
	}
}

// TestFrame_BackgroundPatternActive_StartsPrimaryAndTogglesWithoutDrift
// references AC-4 (a newly constructed frame reports the primary pattern
// active) and AC-3 (the flip alternates primary/alternate with no
// accumulated drift across repeated invocations). Exercised on a
// zero-value frame -- no GTK display required (Test Notes).
func TestFrame_BackgroundPatternActive_StartsPrimaryAndTogglesWithoutDrift(t *testing.T) {
	f := &frame{}

	if f.backgroundPatternActive() {
		t.Fatalf("newly constructed frame reports the alternate pattern active, want primary (AC-4)")
	}

	wantAfterFlip := []bool{true, false, true, false}
	for i, want := range wantAfterFlip {
		f.flipBackgroundPattern()
		if got := f.backgroundPatternActive(); got != want {
			t.Errorf("after flip #%d: backgroundPatternActive() = %v, want %v (AC-3)", i+1, got, want)
		}
	}
}

// TestFrame_BackgroundPatternStyle_TogglingSwapsOnlyBackground references
// AC-5: toggling produces a stylesheet whose window background is the
// alternate value, toggling back restores the primary value, and every
// other emitted declaration is unchanged. Tests the state-plus-selection
// composition directly against buildStylesheet rather than driving GTK
// (Test Notes).
func TestFrame_BackgroundPatternStyle_TogglingSwapsOnlyBackground(t *testing.T) {
	f := &frame{windowStyle: MergeStyle(WindowStyleDefaults, nil)}

	primaryCSS := buildStylesheet(f.backgroundPatternStyle(), nil)
	wantPrimary := fmt.Sprintf("background-color: %s;", f.windowStyle["background"])
	if !strings.Contains(primaryCSS, wantPrimary) {
		t.Fatalf("primary stylesheet = %q, want it to contain %q", primaryCSS, wantPrimary)
	}

	f.flipBackgroundPattern()
	altCSS := buildStylesheet(f.backgroundPatternStyle(), nil)
	wantAlt := fmt.Sprintf("background-color: %s;", f.windowStyle["background_alt"])
	if !strings.Contains(altCSS, wantAlt) {
		t.Errorf("alternate stylesheet = %q, want it to contain %q", altCSS, wantAlt)
	}
	if stripBackgroundColorLine(primaryCSS) != stripBackgroundColorLine(altCSS) {
		t.Errorf("stylesheets differ beyond the background-color declaration:\nprimary=%q\nalt=%q", primaryCSS, altCSS)
	}

	f.flipBackgroundPattern()
	restoredCSS := buildStylesheet(f.backgroundPatternStyle(), nil)
	if restoredCSS != primaryCSS {
		t.Errorf("toggling back = %q, want it to restore the primary stylesheet %q", restoredCSS, primaryCSS)
	}
}

// stripBackgroundColorLine removes every "background-color:" declaration
// line from css, isolating comparison to every other emitted declaration
// (TestFrame_BackgroundPatternStyle_TogglingSwapsOnlyBackground, AC-5).
func stripBackgroundColorLine(css string) string {
	lines := strings.Split(css, "\n")
	kept := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "background-color:") {
			continue
		}
		kept = append(kept, l)
	}
	return strings.Join(kept, "\n")
}

// TestFrame_BackgroundPatternStyle_SettingsReapplicationKeepsActivePattern
// references AC-6: re-applying window settings (the config-reload path)
// while the alternate pattern is active leaves the alternate pattern
// active, and the newly supplied alternate value is what the stylesheet
// then carries. applySettings' own body does nothing more than reassign
// f.windowStyle via MergeStyle before restyling, so that reassignment is
// simulated directly here rather than calling the GTK-dependent
// applySettings/restyle methods (Test Notes).
func TestFrame_BackgroundPatternStyle_SettingsReapplicationKeepsActivePattern(t *testing.T) {
	f := &frame{windowStyle: MergeStyle(WindowStyleDefaults, nil)}
	f.flipBackgroundPattern() // alternate now active

	f.windowStyle = MergeStyle(WindowStyleDefaults, map[string]string{"background_alt": "#000000cc"})

	if !f.backgroundPatternActive() {
		t.Fatalf("re-applying window settings cleared the active pattern, want it to remain alternate (AC-6)")
	}

	css := buildStylesheet(f.backgroundPatternStyle(), nil)
	want := "background-color: #000000cc;"
	if !strings.Contains(css, want) {
		t.Errorf("stylesheet = %q, want it to contain the newly supplied alternate value %q", css, want)
	}
}

// TestFrame_FlipBackgroundPattern_DoesNotTouchConfigDerivedStyleMap
// references AC-7 (no config write occurs as a result of toggling),
// asserted structurally per Test Notes: the toggle path never writes into
// f.windowStyle, the config-derived merged map that applySettings produces
// from config -- so nothing in the toggle path could reach the config file.
func TestFrame_FlipBackgroundPattern_DoesNotTouchConfigDerivedStyleMap(t *testing.T) {
	f := &frame{windowStyle: MergeStyle(WindowStyleDefaults, nil)}
	before := make(map[string]string, len(f.windowStyle))
	for k, v := range f.windowStyle {
		before[k] = v
	}

	f.flipBackgroundPattern()
	f.flipBackgroundPattern()
	f.flipBackgroundPattern()

	if len(f.windowStyle) != len(before) {
		t.Fatalf("toggling changed the config-derived style map size: got %d keys, want %d", len(f.windowStyle), len(before))
	}
	for k, want := range before {
		if f.windowStyle[k] != want {
			t.Errorf("toggling wrote into the config-derived style map[%q] = %q, want unchanged %q (AC-7)", k, f.windowStyle[k], want)
		}
	}
}
