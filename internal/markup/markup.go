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

// allowedTags is the whitelist of Pango markup element names. Attribute
// values are not validated; Pango ignores attributes it does not
// understand, so only well-formedness and the element name matter here.
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
