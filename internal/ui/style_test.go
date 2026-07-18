package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

// --- Section row-packing layout (task0001) ---

// TestBuildStylesheet_NoLongerContainsFlowBoxChildRule references AC-3: the
// FlowBox container (and its auto-inserted "flowboxchild" wrapper node) is
// gone, so the assembled stylesheet no longer contains a flowboxchild rule.
func TestBuildStylesheet_NoLongerContainsFlowBoxChildRule(t *testing.T) {
	windowStyle := MergeStyle(WindowStyleDefaults, nil)

	css := buildStylesheet(windowStyle, nil)

	if strings.Contains(css, "flowboxchild") {
		t.Errorf("buildStylesheet() still contains a flowboxchild rule: %q", css)
	}
}

// TestSectionGapConstants_MatchDesignTokens references task0001 AC-3: the
// packing gap is 1 unit + divider width + 1 unit (13px, the on-screen
// footprint of a vdivider plus its margins), and the row gap stays zero —
// row boxes carry zero intrinsic spacing, the vertical distance between
// rows comes entirely from the hdivider widget's own height.
func TestSectionGapConstants_MatchDesignTokens(t *testing.T) {
	want := spacingUnit + dividerWidth + spacingUnit
	if sectionColumnGap != want {
		t.Errorf("sectionColumnGap = %d, want %d (unit + divider width + unit)", sectionColumnGap, want)
	}
	if sectionColumnGap != 13 {
		t.Errorf("sectionColumnGap = %d, want 13", sectionColumnGap)
	}
	if sectionRowGap != 0 {
		t.Errorf("sectionRowGap = %d, want 0", sectionRowGap)
	}
}

// --- Divider rules + banner spacing (task0001) ---

// TestDividerConstants_MatchDesignIntent references AC-1/AC-2: the divider
// color and thickness are fixed constants matching the design's
// color.divider / border.divider-width values, and the banner-to-sections
// gap is exactly 1 spacing unit (AC-4).
func TestDividerConstants_MatchDesignIntent(t *testing.T) {
	if dividerColor != "#666666" {
		t.Errorf("dividerColor = %q, want %q", dividerColor, "#666666")
	}
	if dividerWidth != 1 {
		t.Errorf("dividerWidth = %d, want 1", dividerWidth)
	}
	if bannerSectionGap != spacingUnit {
		t.Errorf("bannerSectionGap = %d, want %d (1 spacing unit)", bannerSectionGap, spacingUnit)
	}
}

// TestWindowCSS_IncludesHorizontalDividerRule references AC-1: the window
// stylesheet contains a dv-hdivider rule with the divider color, 1px
// height, 6px horizontal margins and zero vertical margins.
func TestWindowCSS_IncludesHorizontalDividerRule(t *testing.T) {
	style := MergeStyle(WindowStyleDefaults, nil)
	css := windowCSS(style)

	if !strings.Contains(css, ".dv-hdivider {") {
		t.Fatalf("windowCSS() missing .dv-hdivider rule: %q", css)
	}
	for _, want := range []string{
		fmt.Sprintf("background-color: %s;", dividerColor),
		fmt.Sprintf("min-height: %dpx;", dividerWidth),
		fmt.Sprintf("margin: 0 %dpx;", spacingUnit),
	} {
		if !strings.Contains(css, want) {
			t.Errorf("windowCSS() = %q, want it to contain %q", css, want)
		}
	}
}

// TestWindowCSS_IncludesVerticalDividerRule references AC-2: the window
// stylesheet contains a dv-vdivider rule with the divider color, 1px width
// and 6px margins.
func TestWindowCSS_IncludesVerticalDividerRule(t *testing.T) {
	style := MergeStyle(WindowStyleDefaults, nil)
	css := windowCSS(style)

	if !strings.Contains(css, ".dv-vdivider {") {
		t.Fatalf("windowCSS() missing .dv-vdivider rule: %q", css)
	}
	for _, want := range []string{
		fmt.Sprintf("background-color: %s;", dividerColor),
		fmt.Sprintf("min-width: %dpx;", dividerWidth),
		fmt.Sprintf("margin: %dpx;", spacingUnit),
	} {
		if !strings.Contains(css, want) {
			t.Errorf("windowCSS() = %q, want it to contain %q", css, want)
		}
	}
}

// TestWindowCSS_BannerHasBottomSpacingTowardSections references AC-4: the
// banner styling includes 1 unit (6px) of bottom spacing toward the
// sections.
func TestWindowCSS_BannerHasBottomSpacingTowardSections(t *testing.T) {
	style := MergeStyle(WindowStyleDefaults, nil)
	css := windowCSS(style)

	want := fmt.Sprintf("margin-bottom: %dpx;", bannerSectionGap)
	if !strings.Contains(css, want) {
		t.Errorf("windowCSS() = %q, want it to contain %q", css, want)
	}
}

// --- SPEC key -> internal key translation (rework round 1, task0006) ---

// TestMergeStyle_SectionSpecKeysChangeOutputFontAndColor references AC-1:
// the SPEC schema example {font: "monospace 12", color: "#ffffff"} changes
// the rendered output font/color away from the defaults.
func TestMergeStyle_SectionSpecKeysChangeOutputFontAndColor(t *testing.T) {
	got := MergeStyle(SectionStyleDefaults, map[string]string{
		"font":  "serif 14",
		"color": "#ffffff",
	})

	if got["output-font"] != "serif" {
		t.Errorf("output-font = %q, want %q (translated from SPEC font key)", got["output-font"], "serif")
	}
	if got["output-size"] != "14pt" {
		t.Errorf("output-size = %q, want %q (translated from SPEC font key)", got["output-size"], "14pt")
	}
	if got["output-color"] != "#ffffff" {
		t.Errorf("output-color = %q, want %q (translated from SPEC color key)", got["output-color"], "#ffffff")
	}
	// Label styling is untouched by section-level SPEC keys.
	if got["label-color"] != SectionStyleDefaults["label-color"] {
		t.Errorf("label-color = %q, want default %q (SPEC keys must not affect label styling)", got["label-color"], SectionStyleDefaults["label-color"])
	}
}

// TestMergeStyle_WindowSpecKeysChangeFrameColorAndWidth references AC-2:
// the SPEC schema example {border_color: "#ff0000", border_width: 2}
// changes the frame color and width values produced by the style logic.
func TestMergeStyle_WindowSpecKeysChangeFrameColorAndWidth(t *testing.T) {
	got := MergeStyle(WindowStyleDefaults, map[string]string{
		"border_color": "#ff0000",
		"border_width": "2",
	})

	if got["frame"] != "#ff0000" {
		t.Errorf("frame = %q, want %q (translated from SPEC border_color key)", got["frame"], "#ff0000")
	}
	if got["frame-width"] != "2px" {
		t.Errorf("frame-width = %q, want %q (translated from SPEC border_width key)", got["frame-width"], "2px")
	}
}

// TestMergeStyle_WindowSpecBackgroundKeyAppliesDirectly checks that the
// window "background" SPEC key needs no alias translation: the SPEC name
// and the internal name are identical (AC-2).
func TestMergeStyle_WindowSpecBackgroundKeyAppliesDirectly(t *testing.T) {
	got := MergeStyle(WindowStyleDefaults, map[string]string{"background": "#000000ff"})

	if got["background"] != "#000000ff" {
		t.Errorf("background = %q, want %q", got["background"], "#000000ff")
	}
}

// TestNormalizeLength_NumericOnlyBecomesPixels references AC-3: bare
// numbers are interpreted as pixels; values that already carry a unit (or
// anything unparsable as a bare number) pass through unchanged.
func TestNormalizeLength_NumericOnlyBecomesPixels(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "bare integer", in: "2", want: "2px"},
		{name: "bare decimal", in: "1.5", want: "1.5px"},
		{name: "zero", in: "0", want: "0px"},
		{name: "already has px unit", in: "3px", want: "3px"},
		{name: "already has other unit", in: "0.5em", want: "0.5em"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeLength(tc.in); got != tc.want {
				t.Errorf("normalizeLength(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestMergeStyle_SpecKeyWinsOverConflictingInternalKey references AC-4:
// when both a SPEC key and its internal equivalent are present, the SPEC
// key wins.
func TestMergeStyle_SpecKeyWinsOverConflictingInternalKey(t *testing.T) {
	windowGot := MergeStyle(WindowStyleDefaults, map[string]string{
		"frame":        "#111111",
		"border_color": "#ff0000",
	})
	if windowGot["frame"] != "#ff0000" {
		t.Errorf("frame = %q, want %q (SPEC border_color must win over internal frame)", windowGot["frame"], "#ff0000")
	}

	sectionGot := MergeStyle(SectionStyleDefaults, map[string]string{
		"output-color": "#111111",
		"color":        "#ffffff",
	})
	if sectionGot["output-color"] != "#ffffff" {
		t.Errorf("output-color = %q, want %q (SPEC color must win over internal output-color)", sectionGot["output-color"], "#ffffff")
	}
}

// TestMergeStyle_ExistingInternalKeysStillWorkWithoutSpecEquivalent
// references AC-4: internal keys with no SPEC alias present keep working
// unchanged.
func TestMergeStyle_ExistingInternalKeysStillWorkWithoutSpecEquivalent(t *testing.T) {
	got := MergeStyle(WindowStyleDefaults, map[string]string{"radius": "0px"})
	if got["radius"] != "0px" {
		t.Errorf("radius = %q, want %q", got["radius"], "0px")
	}

	gotSection := MergeStyle(SectionStyleDefaults, map[string]string{"label-color": "#ffffff"})
	if gotSection["label-color"] != "#ffffff" {
		t.Errorf("label-color = %q, want %q", gotSection["label-color"], "#ffffff")
	}
}

// TestMergeStyle_EmptySpecValueFallsBackToDefault covers the "SPEC key with
// empty value" edge row: an empty override must not clobber the default
// with an empty (invalid) CSS value.
func TestMergeStyle_EmptySpecValueFallsBackToDefault(t *testing.T) {
	got := MergeStyle(WindowStyleDefaults, map[string]string{"border_color": "", "border_width": ""})

	if got["frame"] != WindowStyleDefaults["frame"] {
		t.Errorf("frame = %q, want default %q when border_color is empty", got["frame"], WindowStyleDefaults["frame"])
	}
	if got["frame-width"] != WindowStyleDefaults["frame-width"] {
		t.Errorf("frame-width = %q, want default %q when border_width is empty", got["frame-width"], WindowStyleDefaults["frame-width"])
	}
}

// TestMergeStyle_EmptyStyleMapReturnsDefaults covers the "empty style map"
// edge row for both window and section defaults.
func TestMergeStyle_EmptyStyleMapReturnsDefaults(t *testing.T) {
	gotWindow := MergeStyle(WindowStyleDefaults, map[string]string{})
	for k, want := range WindowStyleDefaults {
		if gotWindow[k] != want {
			t.Errorf("MergeStyle(WindowStyleDefaults, {})[%q] = %q, want default %q", k, gotWindow[k], want)
		}
	}

	gotSection := MergeStyle(SectionStyleDefaults, map[string]string{})
	for k, want := range SectionStyleDefaults {
		if gotSection[k] != want {
			t.Errorf("MergeStyle(SectionStyleDefaults, {})[%q] = %q, want default %q", k, gotSection[k], want)
		}
	}
}

// TestParseFontDescription_UnparsableFallsBackWithoutError covers the
// "unparsable font string" edge row: a font description without a
// trailing bare-number size is reported as unparsable rather than
// producing a garbage size, and MergeStyle then leaves the defaults in
// place instead of erroring.
func TestParseFontDescription_UnparsableFallsBackWithoutError(t *testing.T) {
	cases := []string{"monospace", "monospace bold", ""}
	for _, in := range cases {
		t.Run(fmt.Sprintf("%q", in), func(t *testing.T) {
			if _, _, ok := parseFontDescription(in); ok {
				t.Errorf("parseFontDescription(%q) reported ok=true, want unparsable", in)
			}
		})
	}

	got := MergeStyle(SectionStyleDefaults, map[string]string{"font": "monospace"})
	if got["output-font"] != SectionStyleDefaults["output-font"] {
		t.Errorf("output-font = %q, want default %q for unparsable font string", got["output-font"], SectionStyleDefaults["output-font"])
	}
	if got["output-size"] != SectionStyleDefaults["output-size"] {
		t.Errorf("output-size = %q, want default %q for unparsable font string", got["output-size"], SectionStyleDefaults["output-size"])
	}
}

// TestParseFontDescription_SplitsFamilyAndSize checks multi-word families
// split correctly, with the trailing bare number becoming a pt size.
func TestParseFontDescription_SplitsFamilyAndSize(t *testing.T) {
	family, size, ok := parseFontDescription("DejaVu Sans Mono 10")
	if !ok {
		t.Fatalf("parseFontDescription() reported unparsable for a valid description")
	}
	if family != "DejaVu Sans Mono" {
		t.Errorf("family = %q, want %q", family, "DejaVu Sans Mono")
	}
	if size != "10pt" {
		t.Errorf("size = %q, want %q", size, "10pt")
	}
}

// --- Drift guard (AC-5): defaults table vs design-system/tokens.yaml ---

type tokenEntry struct {
	Value string `yaml:"value"`
}

type typographyValue struct {
	Family string `yaml:"family"`
	Size   string `yaml:"size"`
}

type typographyEntry struct {
	Value typographyValue `yaml:"value"`
}

type designTokens struct {
	Color struct {
		Surface      tokenEntry `yaml:"surface"`
		OnSurface    tokenEntry `yaml:"on-surface"`
		OnSurfaceDim tokenEntry `yaml:"on-surface-dim"`
		Frame        tokenEntry `yaml:"frame"`
		Error        tokenEntry `yaml:"error"`
		ErrorSurface tokenEntry `yaml:"error-surface"`
	} `yaml:"color"`
	Typography struct {
		Output       typographyEntry `yaml:"output"`
		SectionLabel typographyEntry `yaml:"section-label"`
	} `yaml:"typography"`
	Spacing struct {
		Unit tokenEntry `yaml:"unit"`
	} `yaml:"spacing"`
	Radius struct {
		Window tokenEntry `yaml:"window"`
	} `yaml:"radius"`
	Border struct {
		FrameWidth tokenEntry `yaml:"frame-width"`
	} `yaml:"border"`
}

// TestStyleDefaults_MatchDesignTokens references AC-5: the ui defaults
// table must match design-system/tokens.yaml so token/runtime drift fails
// the test suite. go test sets the working directory to the package
// directory, so the path below reaches the repository root from
// internal/ui.
func TestStyleDefaults_MatchDesignTokens(t *testing.T) {
	path := filepath.Join("..", "..", "design-system", "tokens.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var tokens designTokens
	if err := yaml.Unmarshal(data, &tokens); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	wantWindow := map[string]string{
		"background":       tokens.Color.Surface.Value,
		"frame":            tokens.Color.Frame.Value,
		"frame-width":      tokens.Border.FrameWidth.Value,
		"radius":           tokens.Radius.Window.Value,
		"error-color":      tokens.Color.Error.Value,
		"error-background": tokens.Color.ErrorSurface.Value,
	}
	for k, want := range wantWindow {
		if want == "" {
			t.Fatalf("tokens.yaml did not yield a value for window key %q; check the struct tags", k)
		}
		if got := WindowStyleDefaults[k]; got != want {
			t.Errorf("WindowStyleDefaults[%q] = %q, want %q (design-system/tokens.yaml)", k, got, want)
		}
	}

	wantSection := map[string]string{
		"label-color":  tokens.Color.OnSurfaceDim.Value,
		"label-font":   tokens.Typography.SectionLabel.Value.Family,
		"label-size":   tokens.Typography.SectionLabel.Value.Size,
		"output-color": tokens.Color.OnSurface.Value,
		"output-font":  tokens.Typography.Output.Value.Family,
		"output-size":  tokens.Typography.Output.Value.Size,
	}
	for k, want := range wantSection {
		if want == "" {
			t.Fatalf("tokens.yaml did not yield a value for section key %q; check the struct tags", k)
		}
		if got := SectionStyleDefaults[k]; got != want {
			t.Errorf("SectionStyleDefaults[%q] = %q, want %q (design-system/tokens.yaml)", k, got, want)
		}
	}
}
