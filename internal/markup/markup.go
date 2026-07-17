// Package markup decides whether text is valid Pango markup and, when it is
// not, produces an escaped plain-text fallback. Validation is performed
// without GTK/Pango (NFR3): the input is parsed as an XML fragment and
// checked against the whitelist of Pango markup element names.
package markup

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

// allowedTags is the whitelist of Pango markup element names.
var allowedTags = map[string]bool{
	"span":   true,
	"b":      true,
	"i":      true,
	"u":      true,
	"s":      true,
	"tt":     true,
	"sub":    true,
	"sup":    true,
	"small":  true,
	"big":    true,
	"markup": true,
}

// allowedSpanAttrs is the whitelist of attribute names Pango accepts on
// a <span> element. Attribute VALUES are not validated (documented
// limitation: Pango's value parsing is not reproducible without GTK,
// and value errors are rare); only the attribute NAME is checked, so
// GTK never receives a <span> attribute it would reject outright.
// Every other whitelisted tag takes no attributes at all.
var allowedSpanAttrs = map[string]bool{
	"font": true, "font_desc": true, "font_family": true, "face": true,
	"size": true, "style": true, "weight": true, "variant": true, "stretch": true,
	"foreground": true, "fgcolor": true, "color": true,
	"background": true, "bgcolor": true,
	"alpha": true, "fgalpha": true,
	"background_alpha": true, "bgalpha": true,
	"underline": true, "underline_color": true,
	"overline": true, "overline_color": true,
	"rise": true, "baseline_shift": true, "font_scale": true,
	"strikethrough": true, "strikethrough_color": true,
	"fallback": true, "lang": true, "letter_spacing": true,
	"gravity": true, "gravity_hint": true, "font_features": true,
	"allow_breaks": true, "insert_hyphens": true, "line_height": true,
	"text_transform": true, "segment": true, "show": true,
}

// Render decides whether text is valid Pango markup. When it is, text is
// returned unchanged together with true. Otherwise text is returned escaped
// (safe to display as plain text) together with false. The returned string
// is always safe to feed to a Pango markup parser.
func Render(text string) (string, bool) {
	if isValidMarkup(text) {
		return text, true
	}
	return escape(text), false
}

// isValidMarkup reports whether text is a well-formed XML fragment made up
// exclusively of elements in allowedTags. text is wrapped in a synthetic
// root element so that it need not itself be a single well-formed document
// (multiple top-level tags/text runs are fine, matching Pango's markup
// grammar).
func isValidMarkup(text string) bool {
	dec := xml.NewDecoder(strings.NewReader("<markup-root>" + text + "</markup-root>"))

	depth := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return true
		}
		if err != nil {
			return false
		}

		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 1 {
				// The synthetic root; not part of the user's input.
				continue
			}
			if !allowedTags[t.Name.Local] {
				return false
			}
			if t.Name.Local == "span" {
				for _, attr := range t.Attr {
					if !allowedSpanAttrs[attr.Name.Local] {
						return false
					}
				}
			} else if len(t.Attr) > 0 {
				// Every other whitelisted tag takes no attributes.
				return false
			}
		case xml.EndElement:
			depth--
		}
	}
}

// escape returns text with the XML/Pango special characters (<, >, &, ', ")
// escaped, so the result can never be misparsed as markup.
func escape(text string) string {
	var buf bytes.Buffer
	// xml.EscapeText cannot fail; bytes.Buffer never returns a write error.
	_ = xml.EscapeText(&buf, []byte(text))
	return buf.String()
}
