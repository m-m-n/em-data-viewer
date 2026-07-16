package markup

import (
	"encoding/xml"
	"io"
	"os"
	"strings"
	"testing"
)

// decodeXMLText wraps s in a synthetic root and concatenates all decoded
// character data. It is used to prove that an escaped string is itself
// well-formed XML text that round-trips back to the original content --
// i.e. it contains no unescaped '<', '>' or '&' outside of entities.
func decodeXMLText(t *testing.T, s string) string {
	t.Helper()

	dec := xml.NewDecoder(strings.NewReader("<r>" + s + "</r>"))
	var out strings.Builder
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("escaped output %q is not valid XML text: %v", s, err)
		}
		if cd, ok := tok.(xml.CharData); ok {
			out.Write(cd)
		}
	}
	return out.String()
}

// TestRender_ValidMarkup covers AC-1 and AC-2: plain text and text using
// only whitelisted Pango tags passes through unchanged, reported valid.
func TestRender_ValidMarkup(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"plain text no special chars", "hello world"},
		{"multi-line plain text", "line one\nline two\nline three"},
		{"single whitelisted tag", "<b>bold</b>"},
		{"whitelisted tag with attributes", `<span foreground="red" size="large">text</span>`},
		{"multiple sibling whitelisted tags", "<b>bold</b> and <i>italic</i>"},
		{"deeply nested whitelisted tags", "<span><span><span>text</span></span></span>"},
		{"all whitelisted tag names", "<span><b><i><u><s><tt><sub><sup><small><big><markup>x</markup></big></small></sup></sub></tt></s></u></i></b></span>"},
		{"already-escaped ampersand entity", "Tom &amp; Jerry"},
		{"numeric character reference", "&#960;"},
		{"hex character reference", "&#x3c0;"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, valid := Render(tc.input)
			if !valid {
				t.Fatalf("Render(%q) valid = false, want true", tc.input)
			}
			if got != tc.input {
				t.Fatalf("Render(%q) = %q, want unchanged input", tc.input, got)
			}
		})
	}
}

// TestRender_MalformedInput covers AC-3: unclosed tags, bad entities, and
// raw '<'/'&' that break parsing all fall back to the escaped input,
// reported invalid.
func TestRender_MalformedInput(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"unclosed tag", "<b>bold"},
		{"mismatched closing tag", "<b>bold</i>"},
		{"unknown named entity", "Fish &foo; Chips"},
		{"bare ampersand not an entity", "Fish & Chips"},
		{"bare less-than breaking parse", "5 < 3 is true"},
		{"raw less-than inside quoted attribute", `<span foreground="<red>">text</span>`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, valid := Render(tc.input)
			if valid {
				t.Fatalf("Render(%q) valid = true, want false", tc.input)
			}
			want := escape(tc.input)
			if got != want {
				t.Fatalf("Render(%q) = %q, want escaped %q", tc.input, got, want)
			}
		})
	}
}

// TestRender_NonWhitelistedElements covers AC-4: well-formed input using
// elements outside the Pango whitelist falls back to the escaped input,
// reported invalid.
func TestRender_NonWhitelistedElements(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"single non-whitelisted element", "<script>alert(1)</script>"},
		{"non-whitelisted element nested in whitelisted one", "<b>bold <foo>bar</foo></b>"},
		{"well-formed but unknown root element", "<div>text</div>"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, valid := Render(tc.input)
			if valid {
				t.Fatalf("Render(%q) valid = true, want false", tc.input)
			}
			want := escape(tc.input)
			if got != want {
				t.Fatalf("Render(%q) = %q, want escaped %q", tc.input, got, want)
			}
		})
	}
}

// TestRender_EscapedOutputIsFullyEscaped covers AC-5: for every invalid
// input, the escaped output must contain no unescaped '<', '>' or '&'
// outside of entities. This is proven by round-tripping the escaped output
// back through an XML decoder and checking it reconstructs the original
// text exactly.
func TestRender_EscapedOutputIsFullyEscaped(t *testing.T) {
	inputs := []string{
		"<b>bold",
		"Fish &foo; Chips",
		"Fish & Chips",
		"5 < 3 is true",
		`<span foreground="<red>">text</span>`,
		"<script>alert(1)</script>",
		"<b>bold <foo>bar</foo></b>",
		"raw <, >, & all at once",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			got, valid := Render(input)
			if valid {
				t.Fatalf("Render(%q) valid = true, want false (input is malformed/non-whitelisted)", input)
			}
			if strings.ContainsAny(got, "<>") {
				t.Fatalf("escaped output %q still contains raw '<' or '>'", got)
			}
			roundTripped := decodeXMLText(t, got)
			if roundTripped != input {
				t.Fatalf("escaped output %q round-trips to %q, want original %q", got, roundTripped, input)
			}
		})
	}
}

// TestPackage_DoesNotImportGotk4 covers AC-6: the markup package must not
// import gotk4, so it (and go test ./... for it) works headlessly.
func TestPackage_DoesNotImportGotk4(t *testing.T) {
	src, err := os.ReadFile("markup.go")
	if err != nil {
		t.Fatalf("reading markup.go: %v", err)
	}
	if strings.Contains(string(src), "gotk4") {
		t.Fatalf("markup.go must not import gotk4 (NFR3)")
	}
}
