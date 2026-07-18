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

// sectionsView manages the greedily row-packed sections inside the window:
// sections share a row left-to-right in config order while width allows,
// wrapping to the next row when it does not (FR1, FR2; no divider lines,
// no column alignment). widget is a vertical box; each element of rows is
// one horizontal row box currently appended to it, holding a contiguous
// run of section roots in config order.
type sectionsView struct {
	widget *gtk.Box
	rows   []*gtk.Box

	order  []string
	byKey  map[string]*sectionWidgets
	nextID int

	// lastAssignment is the row assignment actually rendered as rows, kept
	// to skip rebuilding row widgets when reflow recomputes the same
	// layout (FR4, AC-5).
	lastAssignment [][]int
}

func newSectionsView() *sectionsView {
	widget := gtk.NewBox(gtk.OrientationVertical, sectionRowGap)

	return &sectionsView{
		widget: widget,
		byKey:  make(map[string]*sectionWidgets),
	}
}

// setSections replaces the displayed sections with sections, in the given
// order (AC-4), and reflows them into rows (FR4).
func (sv *sectionsView) setSections(sections []SectionViewModel) {
	for _, row := range sv.rows {
		sv.widget.Remove(row)
	}
	sv.rows = nil
	sv.lastAssignment = nil

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

		sv.byKey[s.Key] = &sectionWidgets{
			class:  class,
			style:  MergeStyle(SectionStyleDefaults, s.Style),
			root:   root,
			label:  label,
			output: output,
		}
		sv.order = append(sv.order, s.Key)
	}

	sv.reflow()
}

// updateSection replaces exactly the targeted section's content (AC-4).
// isError renders message in the error style with an "error: " prefix
// instead of treating it as pre-validated Pango markup (AC-6). Unknown
// keys are ignored (the section may have been removed by a later reload).
// A content change can change the section's natural width, so this also
// triggers a reflow (FR4).
func (sv *sectionsView) updateSection(key, markup string, isError bool) {
	sw, ok := sv.byKey[key]
	if !ok {
		return
	}
	if isError {
		sw.output.SetUseMarkup(false)
		sw.output.SetText(fmt.Sprintf("error: %s", markup))
		sw.output.AddCSSClass("dv-error")
	} else {
		sw.output.RemoveCSSClass("dv-error")
		sw.output.SetUseMarkup(true)
		sw.output.SetMarkup(markup)
	}
	sv.reflow()
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

// reflow recomputes the row assignment from every section root's natural
// horizontal width (for_size unconstrained) and the container's currently
// allocated width, then rebuilds the row widgets only when the assignment
// actually changed (FR3, FR4, AC-5). Safe to call with zero sections, and
// before the window has ever been allocated a width (AllocatedWidth is then
// 0, so packRows degrades to one section per row until a later reflow —
// e.g. the window's map signal — corrects it).
func (sv *sectionsView) reflow() {
	widths := make([]int, len(sv.order))
	for i, key := range sv.order {
		_, natural, _, _ := sv.byKey[key].root.Measure(gtk.OrientationHorizontal, -1)
		widths[i] = natural
	}

	assignment := packRows(widths, sectionColumnGap, sv.widget.AllocatedWidth())
	if assignmentsEqual(assignment, sv.lastAssignment) {
		return
	}

	sv.rebuildRows(assignment)
	sv.lastAssignment = assignment
}

// rebuildRows re-parents every section root into fresh row boxes per
// assignment: it detaches each root from wherever it is currently parented
// (a freshly built section, not yet parented anywhere, is left alone),
// drops the old row boxes from the container, then builds and appends new
// row boxes with the roots in assignment order. Postcondition: every root
// is parented to exactly one row box; row count equals len(assignment).
func (sv *sectionsView) rebuildRows(assignment [][]int) {
	for _, key := range sv.order {
		root := sv.byKey[key].root
		if root.Parent() != nil {
			root.Unparent()
		}
	}
	for _, row := range sv.rows {
		sv.widget.Remove(row)
	}

	rows := make([]*gtk.Box, len(assignment))
	for i, group := range assignment {
		row := gtk.NewBox(gtk.OrientationHorizontal, sectionColumnGap)
		for _, idx := range group {
			row.Append(sv.byKey[sv.order[idx]].root)
		}
		sv.widget.Append(row)
		rows[i] = row
	}
	sv.rows = rows
}

// packRows greedily assigns widths (natural section widths, in config
// order) to rows: each row accumulates items left-to-right until the next
// item would not fit (used + gap + width > avail), then wraps. Every row
// holds at least one item — an item wider than avail still gets its own
// row, and avail <= 0 degrades to one item per row. Empty input yields no
// rows. Pure and deterministic; no widget types involved (FR2).
func packRows(widths []int, gap, avail int) [][]int {
	if len(widths) == 0 {
		return nil
	}

	var rows [][]int
	row := []int{0}
	used := widths[0]

	for i := 1; i < len(widths); i++ {
		w := widths[i]
		if avail > 0 && used+gap+w <= avail {
			row = append(row, i)
			used += gap + w
			continue
		}
		rows = append(rows, row)
		row = []int{i}
		used = w
	}
	rows = append(rows, row)

	return rows
}

// assignmentsEqual reports whether two row assignments are identical: same
// number of rows, and each row holds the same indices in the same order
// (AC-5). Used by reflow to skip a widget rebuild when the newly computed
// layout matches what is already on screen.
func assignmentsEqual(a, b [][]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}
