package ui

import "github.com/diamondburned/gotk4/pkg/gtk/v4"

// banner is the full-width config-error bar pinned above all sections
// (FR9). An empty message hides it (AC-5).
type banner struct {
	widget *gtk.Label
}

func newBanner() *banner {
	label := gtk.NewLabel("")
	label.SetXAlign(0)
	label.SetWrap(false)
	label.AddCSSClass("dv-banner")
	label.SetVisible(false)
	return &banner{widget: label}
}

// setError shows message in the banner, or hides the banner when message
// is empty (AC-5).
func (b *banner) setError(message string) {
	if message == "" {
		b.widget.SetVisible(false)
		b.widget.SetText("")
		return
	}
	b.widget.SetText(message)
	b.widget.SetVisible(true)
}
