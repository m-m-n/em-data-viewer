package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// SectionViewModel is one section's display data, translated by the app
// layer from config.SectionConfig (the ui package stays decoupled from the
// config package). Key is the stable identity from SectionConfig.Key();
// display order is the order of the slice passed to SetSections — Key is
// identity, not order.
type SectionViewModel struct {
	Key   string
	Label string
	Style map[string]string
}

// sectionWidgets holds the live widgets and merged style for one section.
type sectionWidgets struct {
	class  string
	style  map[string]string
	root   *gtk.Box
	label  *gtk.Label
	output *gtk.Label
}

// maxSectionsPerLine bounds GtkFlowBox's per-line child request high enough
// that no realistic section count ever forces a row to wrap on count alone
// — only available width wraps a row (FR2).
const maxSectionsPerLine = 100

// sectionsView manages the horizontally reflowing row of sections inside
// the window: sections share a row left-to-right in config order while
// width allows, wrapping to the next row when it does not (no divider
// lines).
type sectionsView struct {
	widget *gtk.FlowBox
	order  []string
	byKey  map[string]*sectionWidgets
	nextID int
}

func newSectionsView() *sectionsView {
	flow := gtk.NewFlowBox()
	flow.SetOrientation(gtk.OrientationHorizontal)
	flow.SetSelectionMode(gtk.SelectionNone)
	flow.SetHomogeneous(false)
	flow.SetColumnSpacing(sectionColumnGap)
	flow.SetRowSpacing(sectionRowGap)
	flow.SetMinChildrenPerLine(1)
	flow.SetMaxChildrenPerLine(maxSectionsPerLine)

	return &sectionsView{
		widget: flow,
		byKey:  make(map[string]*sectionWidgets),
	}
}

// setSections replaces the displayed sections with sections, appended to
// the flow container left-to-right in the given order (AC-3).
func (sv *sectionsView) setSections(sections []SectionViewModel) {
	for _, key := range sv.order {
		sv.widget.Remove(sv.byKey[key].root)
	}
	sv.order = sv.order[:0]
	sv.byKey = make(map[string]*sectionWidgets, len(sections))

	for _, s := range sections {
		sv.nextID++
		class := fmt.Sprintf("dv-section-%d", sv.nextID)

		label := gtk.NewLabel(s.Label)
		label.SetXAlign(0)
		label.AddCSSClass("dv-section-label")
		label.AddCSSClass(class)
		label.SetVisible(s.Label != "")

		output := gtk.NewLabel("")
		output.SetXAlign(0)
		output.SetWrap(true)
		output.SetUseMarkup(true)
		output.AddCSSClass("dv-output")
		output.AddCSSClass(class)

		root := gtk.NewBox(gtk.OrientationVertical, labelOutputGap)
		root.Append(label)
		root.Append(output)

		sv.widget.Append(root)

		sv.byKey[s.Key] = &sectionWidgets{
			class:  class,
			style:  MergeStyle(SectionStyleDefaults, s.Style),
			root:   root,
			label:  label,
			output: output,
		}
		sv.order = append(sv.order, s.Key)
	}
}

// updateSection replaces exactly the targeted section's content (AC-4).
// isError renders message in the error style with an "error: " prefix
// instead of treating it as pre-validated Pango markup (AC-6). Unknown
// keys are ignored (the section may have been removed by a later reload).
func (sv *sectionsView) updateSection(key, markup string, isError bool) {
	sw, ok := sv.byKey[key]
	if !ok {
		return
	}
	if isError {
		sw.output.SetUseMarkup(false)
		sw.output.SetText(fmt.Sprintf("error: %s", markup))
		sw.output.AddCSSClass("dv-error")
		return
	}
	sw.output.RemoveCSSClass("dv-error")
	sw.output.SetUseMarkup(true)
	sw.output.SetMarkup(markup)
}

// styleEntries returns the per-section style, keyed by each section's
// unique CSS class, for stylesheet rebuilding.
func (sv *sectionsView) styleEntries() []sectionStyleEntry {
	entries := make([]sectionStyleEntry, 0, len(sv.order))
	for _, key := range sv.order {
		sw := sv.byKey[key]
		entries = append(entries, sectionStyleEntry{class: sw.class, style: sw.style})
	}
	return entries
}
