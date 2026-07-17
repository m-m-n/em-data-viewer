// Package ui implements the GTK4 widget: an undecorated frame window with
// stacked sections, drag-to-move, a right-click context menu and a
// config-error banner.
package ui

import (
	"fmt"
	"regexp"
	"strings"
)

// Spacing unit (design-system/tokens.yaml spacing.unit) and its fixed
// multiples. These are layout constants, not user-overridable style values.
const (
	spacingUnit    = 6 // px
	windowPadding  = 2 * spacingUnit
	sectionGap     = 2 * spacingUnit
	labelOutputGap = 1 * spacingUnit
)

// WindowStyleDefaults are the design-token defaults for window-level
// styling (design-system/tokens.yaml). Keys are overridden individually by
// config.Config.Window.Style; keys absent from this table are ignored by
// MergeStyle so unrelated user keys never error.
var WindowStyleDefaults = map[string]string{
	"background":       "#1e1e1eee",
	"frame":            "#888888",
	"frame-width":      "1px",
	"radius":           "8px",
	"error-color":      "#e57373",
	"error-background": "#3a2222ee",
}

// SectionStyleDefaults are the design-token defaults for per-section
// styling. Keys are overridden individually by config.SectionConfig.Style.
var SectionStyleDefaults = map[string]string{
	"label-color":  "#9e9e9e",
	"label-font":   "sans-serif",
	"label-size":   "9pt",
	"output-color": "#e0e0e0",
	"output-font":  "monospace",
	"output-size":  "12pt",
}

// specKeyAliases maps SPEC Config Schema style keys (the public contract
// documented in SPEC.md) onto their internal role-based equivalents. A
// window style map and a section style map never share these key names, so
// one alias table safely serves both contexts: MergeStyle discards any
// translated key that the given defaults table doesn't recognize.
//
// "background" needs no entry: the window SPEC key and the internal key
// are already the same name. "font" is handled separately in
// translateSpecKeys because it splits into two internal keys.
var specKeyAliases = map[string]string{
	"border_color": "frame",
	"border_width": "frame-width",
	"color":        "output-color",
}

// numericOnly matches a bare number (no unit suffix).
var numericOnly = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// normalizeLength interprets a bare number as pixels (AC-3); a value that
// already carries a unit — or anything else that isn't a bare number —
// passes through unchanged.
func normalizeLength(v string) string {
	v = strings.TrimSpace(v)
	if numericOnly.MatchString(v) {
		return v + "px"
	}
	return v
}

// parseFontDescription splits a Pango font description (e.g.
// "monospace 12") into a family and a CSS-ready pt size ("12pt"). The last
// whitespace-separated token must be a bare number (the Pango point size);
// anything else — including a description with no size at all — is
// reported as unparsable (ok == false) so the caller can fall back to the
// defaults instead of producing a bogus value.
func parseFontDescription(desc string) (family, size string, ok bool) {
	fields := strings.Fields(desc)
	if len(fields) < 2 {
		return "", "", false
	}
	last := fields[len(fields)-1]
	if !numericOnly.MatchString(last) {
		return "", "", false
	}
	return strings.Join(fields[:len(fields)-1], " "), last + "pt", true
}

// translateSpecKeys returns a copy of overrides with every recognized SPEC
// Config Schema key additionally written under its internal equivalent.
// The original entries are kept as-is (MergeStyle discards whichever ones
// its defaults table doesn't know), and the SPEC-derived value is applied
// after the copy so it wins when both the SPEC key and the internal key
// are present (AC-4). An empty SPEC value is treated as absent — it falls
// back to the default instead of clobbering it with an empty CSS value —
// and an unparsable font description is dropped without error.
func translateSpecKeys(overrides map[string]string) map[string]string {
	if len(overrides) == 0 {
		return overrides
	}

	translated := make(map[string]string, len(overrides)+2)
	for k, v := range overrides {
		translated[k] = v
	}

	for specKey, internalKey := range specKeyAliases {
		v, ok := overrides[specKey]
		if !ok || v == "" {
			continue
		}
		if specKey == "border_width" {
			v = normalizeLength(v)
		}
		translated[internalKey] = v
	}

	if font, ok := overrides["font"]; ok && font != "" {
		if family, size, ok := parseFontDescription(font); ok {
			translated["output-font"] = family
			translated["output-size"] = size
		}
	}

	return translated
}

// MergeStyle overlays user-provided style values onto the token defaults.
// SPEC Config Schema keys (window border_color/border_width, section
// font/color) are translated onto their internal equivalents first, so
// they take effect the same as the internal keys they alias (AC-1, AC-2,
// AC-4); when both are present the SPEC key wins. Only keys already
// present in defaults are considered; any other key in overrides — SPEC
// alias or not — is ignored without error. The result always contains
// exactly the keys of defaults.
func MergeStyle(defaults, overrides map[string]string) map[string]string {
	translated := translateSpecKeys(overrides)

	merged := make(map[string]string, len(defaults))
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range translated {
		if _, known := defaults[k]; known {
			merged[k] = v
		}
	}
	return merged
}

// windowCSS translates a merged window style map into the GTK CSS block for
// the top-level frame window, the config-error banner and section error
// text (the error/error-background keys are window-scoped: SPEC treats them
// as reserved, single-purpose colors, not per-section values).
func windowCSS(style map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, ".dv-window {\n")
	fmt.Fprintf(&b, "  background-color: %s;\n", style["background"])
	fmt.Fprintf(&b, "  border: %s solid %s;\n", style["frame-width"], style["frame"])
	fmt.Fprintf(&b, "  border-radius: %s;\n", style["radius"])
	fmt.Fprintf(&b, "  padding: %dpx;\n", windowPadding)
	fmt.Fprintf(&b, "}\n")
	fmt.Fprintf(&b, ".dv-sections {\n")
	fmt.Fprintf(&b, "  margin: 0;\n")
	fmt.Fprintf(&b, "}\n")
	fmt.Fprintf(&b, ".dv-banner {\n")
	fmt.Fprintf(&b, "  background-color: %s;\n", style["error-background"])
	fmt.Fprintf(&b, "  color: %s;\n", style["error-color"])
	fmt.Fprintf(&b, "  font-family: monospace;\n")
	fmt.Fprintf(&b, "}\n")
	fmt.Fprintf(&b, ".dv-output.dv-error {\n")
	fmt.Fprintf(&b, "  color: %s;\n", style["error-color"])
	fmt.Fprintf(&b, "}\n")
	return b.String()
}

// sectionCSS translates a merged per-section style map into the GTK CSS
// block for that section's label and output widgets. class is the unique
// per-section CSS class the caller has already assigned to that section's
// widgets, keeping the rule scoped to a single section.
func sectionCSS(class string, style map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, ".%s.dv-section-label {\n", class)
	fmt.Fprintf(&b, "  color: %s;\n", style["label-color"])
	fmt.Fprintf(&b, "  font-family: %s;\n", style["label-font"])
	fmt.Fprintf(&b, "  font-size: %s;\n", style["label-size"])
	fmt.Fprintf(&b, "}\n")
	fmt.Fprintf(&b, ".%s.dv-output {\n", class)
	fmt.Fprintf(&b, "  color: %s;\n", style["output-color"])
	fmt.Fprintf(&b, "  font-family: %s;\n", style["output-font"])
	fmt.Fprintf(&b, "  font-size: %s;\n", style["output-size"])
	fmt.Fprintf(&b, "}\n")
	return b.String()
}

// buildStylesheet assembles the full CSS text applied to the display: the
// window/banner/error rules plus one scoped rule per section.
func buildStylesheet(windowStyle map[string]string, sections []sectionStyleEntry) string {
	var b strings.Builder
	b.WriteString(windowCSS(windowStyle))
	for _, s := range sections {
		b.WriteString(sectionCSS(s.class, s.style))
	}
	return b.String()
}

// sectionStyleEntry pairs a section's assigned CSS class with its merged
// style map, used only to build the stylesheet.
type sectionStyleEntry struct {
	class string
	style map[string]string
}
