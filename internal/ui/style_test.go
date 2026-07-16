package ui

import (
	"fmt"
	"strings"
	"testing"
)

// TestMergeStyle_AppliesOverridesPerKey references AC-1: user config values
// override the token defaults per key.
func TestMergeStyle_AppliesOverridesPerKey(t *testing.T) {
	defaults := map[string]string{
		"background": "#1e1e1eee",
		"frame":      "#888888",
	}

	cases := []struct {
		name      string
		overrides map[string]string
		want      map[string]string
	}{
		{
			name:      "no overrides returns defaults verbatim",
			overrides: nil,
			want: map[string]string{
				"background": "#1e1e1eee",
				"frame":      "#888888",
			},
		},
		{
			name:      "override replaces exactly the matching key",
			overrides: map[string]string{"background": "#000000ff"},
			want: map[string]string{
				"background": "#000000ff",
				"frame":      "#888888",
			},
		},
		{
			name: "override replaces every key when all are given",
			overrides: map[string]string{
				"background": "#111111",
				"frame":      "#222222",
			},
			want: map[string]string{
				"background": "#111111",
				"frame":      "#222222",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MergeStyle(defaults, tc.overrides)
			if len(got) != len(tc.want) {
				t.Fatalf("MergeStyle() = %v, want %v", got, tc.want)
			}
			for k, wantV := range tc.want {
				if got[k] != wantV {
					t.Errorf("MergeStyle()[%q] = %q, want %q", k, got[k], wantV)
				}
			}
		})
	}
}

// TestMergeStyle_UnknownKeysIgnored references AC-1: unknown style keys are
// ignored without error.
func TestMergeStyle_UnknownKeysIgnored(t *testing.T) {
	defaults := map[string]string{"background": "#1e1e1eee"}
	overrides := map[string]string{
		"background":  "#000000ff",
		"typo-kye":    "should be ignored",
		"font-weight": "bold",
	}

	got := MergeStyle(defaults, overrides)

	if len(got) != 1 {
		t.Fatalf("MergeStyle() returned %d keys, want 1 (unknown keys must be ignored): %v", len(got), got)
	}
	if got["background"] != "#000000ff" {
		t.Errorf("MergeStyle()[background] = %q, want override applied", got["background"])
	}
	if _, present := got["typo-kye"]; present {
		t.Errorf("MergeStyle() leaked unknown key %q into result", "typo-kye")
	}
}

// TestMergeStyle_DoesNotMutateInputs guards against a merge that aliases
// the defaults map (a later merge call must not see a previous override).
func TestMergeStyle_DoesNotMutateInputs(t *testing.T) {
	defaults := map[string]string{"background": "#1e1e1eee"}
	_ = MergeStyle(defaults, map[string]string{"background": "#000000ff"})

	if defaults["background"] != "#1e1e1eee" {
		t.Errorf("MergeStyle() mutated the defaults map: got %q", defaults["background"])
	}
}

func TestWindowCSS_TranslatesMergedValues(t *testing.T) {
	style := MergeStyle(WindowStyleDefaults, map[string]string{
		"background": "#000000ff",
		"radius":     "0px",
	})

	css := windowCSS(style)

	for _, want := range []string{
		"background-color: #000000ff;",
		"border-radius: 0px;",
		"border: 1px solid #888888;",
		fmt.Sprintf("padding: %dpx;", windowPadding),
	} {
		if !strings.Contains(css, want) {
			t.Errorf("windowCSS() = %q, want it to contain %q", css, want)
		}
	}
}

func TestWindowCSS_UsesErrorTokensForBannerAndSectionError(t *testing.T) {
	style := MergeStyle(WindowStyleDefaults, nil)

	css := windowCSS(style)

	if !strings.Contains(css, ".dv-banner {") {
		t.Errorf("windowCSS() missing banner rule: %q", css)
	}
	if !strings.Contains(css, "color: #e57373;") {
		t.Errorf("windowCSS() missing error color: %q", css)
	}
	if !strings.Contains(css, "background-color: #3a2222ee;") {
		t.Errorf("windowCSS() missing error banner background: %q", css)
	}
	if !strings.Contains(css, ".dv-output.dv-error {") {
		t.Errorf("windowCSS() missing section-error rule: %q", css)
	}
}

func TestSectionCSS_ScopedToItsOwnClass(t *testing.T) {
	style := MergeStyle(SectionStyleDefaults, map[string]string{"output-color": "#ffffff"})

	css := sectionCSS("dv-section-1", style)

	if !strings.Contains(css, ".dv-section-1.dv-section-label {") {
		t.Errorf("sectionCSS() missing scoped label rule: %q", css)
	}
	if !strings.Contains(css, ".dv-section-1.dv-output {") {
		t.Errorf("sectionCSS() missing scoped output rule: %q", css)
	}
	if !strings.Contains(css, "color: #ffffff;") {
		t.Errorf("sectionCSS() did not apply overridden output color: %q", css)
	}
}

func TestBuildStylesheet_IncludesWindowAndEverySection(t *testing.T) {
	windowStyle := MergeStyle(WindowStyleDefaults, nil)
	sections := []sectionStyleEntry{
		{class: "dv-section-1", style: MergeStyle(SectionStyleDefaults, nil)},
		{class: "dv-section-2", style: MergeStyle(SectionStyleDefaults, map[string]string{"output-color": "#123456"})},
	}

	css := buildStylesheet(windowStyle, sections)

	if !strings.Contains(css, ".dv-window {") {
		t.Errorf("buildStylesheet() missing window rule: %q", css)
	}
	if !strings.Contains(css, ".dv-section-1.dv-output {") {
		t.Errorf("buildStylesheet() missing section 1 rule: %q", css)
	}
	if !strings.Contains(css, ".dv-section-2.dv-output {") {
		t.Errorf("buildStylesheet() missing section 2 rule: %q", css)
	}
	if !strings.Contains(css, "color: #123456;") {
		t.Errorf("buildStylesheet() did not apply section 2's override: %q", css)
	}
}
