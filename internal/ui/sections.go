package ui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// SectionViewModel is one section's display data, translated by the app
// layer from config.SectionConfig (the ui package stays decoupled from the
// config package). Key is the stable identity from SectionConfig.Key();
// display order is the order of the slice passed to SetSections — Key is
// identity, not order. IsBreak marks a forced row-break entry
// (section-row-break task0001 FR2, FR4): when true, Key/Label/Style are
// zero-valued and unused — no widget or style entry is created for it, but
// its position in the slice still forces a row boundary there.
type SectionViewModel struct {
	Key     string
	Label   string
	Style   map[string]string
	IsBreak bool
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

	// rowChildren holds every widget currently appended directly to widget
	// for the current row layout, in append order: row boxes interleaved
	// with the hdividers between them. Tracked separately from rows so a
	// rebuild removes exactly what it added, including the dividers.
	rowChildren []gtk.Widgetter

	order  []string
	byKey  map[string]*sectionWidgets
	nextID int

	// breaks holds the forced-break split points recorded by setSections
	// (section-row-break task0001 FR4), each a position in [0, len(order)]
	// relative to the real (non-break) sections in order — the same
	// coordinate space packRowsWithBreaks expects.
	breaks []int

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
// order (AC-4), and reflows them into rows (FR4). A break entry
// (section-row-break task0001 FR4) creates no widget, no style entry, and
// no order entry; instead its position is recorded in sv.breaks, relative
// to the real sections seen so far, for reflow's row assignment to use.
func (sv *sectionsView) setSections(sections []SectionViewModel) {
	for _, child := range sv.rowChildren {
		sv.widget.Remove(child)
	}
	sv.rows = nil
	sv.rowChildren = nil
	sv.lastAssignment = nil

	sv.order = sv.order[:0]
	sv.byKey = make(map[string]*sectionWidgets, len(sections))
	sv.breaks = sv.breaks[:0]

	for _, s := range sections {
		if s.IsBreak {
			sv.breaks = append(sv.breaks, len(sv.order))
			continue
		}
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
// actually changed (FR3, FR4, AC-5). sv.breaks (section-row-break task0001
// FR4) splits the widths into independently packed segments before the
// usual greedy width packing runs on each; empty otherwise, which makes
// this identical to plain packRows. Safe to call with zero sections, and
// before the window has ever been allocated a width (AllocatedWidth is then
// 0, so width-based packing degrades to one section per row until a later
// reflow — e.g. the window's map signal — corrects it).
func (sv *sectionsView) reflow() {
	widths := make([]int, len(sv.order))
	for i, key := range sv.order {
		_, natural, _, _ := sv.byKey[key].root.Measure(gtk.OrientationHorizontal, -1)
		widths[i] = natural
	}

	assignment := packRowsWithBreaks(widths, sv.breaks, sectionColumnGap, sv.widget.AllocatedWidth())
	if assignmentsEqual(assignment, sv.lastAssignment) {
		return
	}

	sv.rebuildRows(assignment)
	sv.lastAssignment = assignment
}

// rebuildRows re-parents every section root into fresh row boxes per
// assignment, interleaved with fresh divider widgets: it detaches each root
// from wherever it is currently parented (a freshly built section, not yet
// parented anywhere, is left alone), drops the old row boxes and dividers
// from the container, then builds and appends new row boxes with the roots
// in assignment order — an hdivider between adjacent rows, a vdivider
// between adjacent section roots within a row (AC-5). Row boxes carry zero
// intrinsic spacing (AC-3); the on-screen distance comes entirely from the
// divider widgets and their CSS margins. Postcondition: every root is
// parented to exactly one row box; row count equals len(assignment); N rows
// produce N-1 hdividers; a row of M sections produces M-1 vdividers; zero
// sections produce zero rows and zero dividers.
func (sv *sectionsView) rebuildRows(assignment [][]int) {
	for _, key := range sv.order {
		root := sv.byKey[key].root
		if root.Parent() != nil {
			root.Unparent()
		}
	}
	for _, child := range sv.rowChildren {
		sv.widget.Remove(child)
	}

	hCount, vCounts := rowDividerCounts(assignment)

	rows := make([]*gtk.Box, len(assignment))
	children := make([]gtk.Widgetter, 0, len(assignment)+hCount)
	for i, group := range assignment {
		if i > 0 {
			hd := newHDivider()
			sv.widget.Append(hd)
			children = append(children, hd)
		}

		row := gtk.NewBox(gtk.OrientationHorizontal, 0)
		row.Append(sv.byKey[sv.order[group[0]]].root)
		for k := 0; k < vCounts[i]; k++ {
			row.Append(newVDivider())
			row.Append(sv.byKey[sv.order[group[k+1]]].root)
		}
		sv.widget.Append(row)
		children = append(children, row)
		rows[i] = row
	}
	sv.rows = rows
	sv.rowChildren = children
}

// rowDividerCounts derives, from a row assignment, how many divider widgets
// rebuildRows must interleave: N rows yield N-1 horizontal dividers between
// them; a row holding M sections yields M-1 vertical dividers within it;
// zero rows yield zero horizontal dividers (AC-5). Pure and deterministic;
// no widget types involved, so it is testable directly even though the GTK
// widget construction it drives is not (test/README.md).
func rowDividerCounts(assignment [][]int) (horizontal int, vertical []int) {
	if len(assignment) > 1 {
		horizontal = len(assignment) - 1
	}
	vertical = make([]int, len(assignment))
	for i, group := range assignment {
		if len(group) > 1 {
			vertical[i] = len(group) - 1
		}
	}
	return horizontal, vertical
}

// newHDivider returns a stateless horizontal rule widget separating two
// adjacent rows (dv-hdivider); its thickness, color and margins come
// entirely from the window stylesheet (AC-1).
func newHDivider() *gtk.Box {
	d := gtk.NewBox(gtk.OrientationHorizontal, 0)
	d.AddCSSClass("dv-hdivider")
	return d
}

// newVDivider returns a stateless vertical rule widget separating two
// adjacent sections sharing a row (dv-vdivider); its thickness, color and
// margins come entirely from the window stylesheet (AC-2).
func newVDivider() *gtk.Box {
	d := gtk.NewBox(gtk.OrientationVertical, 0)
	d.AddCSSClass("dv-vdivider")
	return d
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

// packRowsWithBreaks packs widths into rows exactly like packRows, except
// forced row breaks (section-row-break task0001 FR4, FR5) split widths into
// independently packed segments first: breaks holds the sorted split
// points, each a position in [0, len(widths)] marking where the current
// row must end regardless of remaining width. Every segment is packed on
// its own with packRows and the resulting rows are concatenated in order,
// with indices translated back into the original widths slice; a natural
// wrap within a segment behaves exactly as packRows alone would (AC-5). A
// split point at 0 or len(widths), or two adjacent split points, yields an
// empty segment and contributes no rows (AC-6). Equivalent to
// packRows(widths, gap, avail) when breaks is empty (AC-7).
func packRowsWithBreaks(widths []int, breaks []int, gap, avail int) [][]int {
	var out [][]int
	start := 0

	appendSegment := func(end int) {
		if end <= start {
			start = end
			return
		}
		for _, row := range packRows(widths[start:end], gap, avail) {
			translated := make([]int, len(row))
			for i, idx := range row {
				translated[i] = idx + start
			}
			out = append(out, translated)
		}
		start = end
	}

	for _, b := range breaks {
		appendSegment(b)
	}
	appendSegment(len(widths))

	return out
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
