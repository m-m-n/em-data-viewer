package markup

import "strings"

// shadeAlpha maps the shade block elements to the foreground alpha used
// when drawing them as a full block.
var shadeAlpha = map[rune]string{
	'░': "25%",
	'▒': "50%",
	'▓': "75%",
}

// ExpandShades replaces the shade characters ░ ▒ ▓ in the text parts of
// the Pango markup s with a full block █ drawn at 25% / 50% / 75%
// foreground alpha. Consecutive identical shades share one <span>. Tags,
// including attribute values, are left untouched. s must be the output of
// Render.
func ExpandShades(s string) string {
	if !strings.ContainsAny(s, "░▒▓") {
		return s
	}

	var b strings.Builder
	inTag := false
	var quote rune
	var run rune // shade character of the currently open span; 0 when none

	closeRun := func() {
		if run != 0 {
			b.WriteString("</span>")
			run = 0
		}
	}

	for _, r := range s {
		if inTag {
			b.WriteRune(r)
			switch {
			case quote != 0:
				if r == quote {
					quote = 0
				}
			case r == '"' || r == '\'':
				quote = r
			case r == '>':
				inTag = false
			}
			continue
		}

		if alpha, ok := shadeAlpha[r]; ok {
			if run != r {
				closeRun()
				b.WriteString(`<span fgalpha="` + alpha + `">`)
				run = r
			}
			b.WriteRune('█')
			continue
		}

		closeRun()
		if r == '<' {
			inTag = true
		}
		b.WriteRune(r)
	}
	closeRun()
	return b.String()
}
