package markup

import "testing"

func TestExpandShades(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"no shade", "abc", "abc"},
		{"light", "░", `<span fgalpha="25%">█</span>`},
		{"medium", "▒", `<span fgalpha="50%">█</span>`},
		{"dark", "▓", `<span fgalpha="75%">█</span>`},
		{"run grouped", "██░░░ 23%", `██<span fgalpha="25%">███</span> 23%`},
		{"mixed run", "░░▒▓▓", `<span fgalpha="25%">██</span><span fgalpha="50%">█</span><span fgalpha="75%">██</span>`},
		{
			"inside span",
			`7d <span foreground="#81c784">█░░ 39%</span>`,
			`7d <span foreground="#81c784">█<span fgalpha="25%">██</span> 39%</span>`,
		},
		{
			"attribute value untouched",
			`<span font_family="░>▒">x░</span>`,
			`<span font_family="░>▒">x<span fgalpha="25%">█</span></span>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExpandShades(tc.input)
			if got != tc.want {
				t.Errorf("ExpandShades(%q) = %q, want %q", tc.input, got, tc.want)
			}
			if !isValidMarkup(got) {
				t.Errorf("ExpandShades(%q) = %q is not valid markup", tc.input, got)
			}
		})
	}
}
