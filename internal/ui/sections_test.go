package ui

import (
	"reflect"
	"testing"
)

// --- packRows (task0001) ---

// TestPackRows references AC-1: greedy row packing — items accumulate into
// a row until the next one would not fit; an exact-fit item stays in the
// row, one unit over wraps to a new row.
func TestPackRows(t *testing.T) {
	cases := []struct {
		name   string
		widths []int
		gap    int
		avail  int
		want   [][]int
	}{
		{
			name:   "width 6 gap 0 widths 1,2,2,2,2,2 packs [1 2 2][2 2 2]",
			widths: []int{1, 2, 2, 2, 2, 2},
			gap:    0,
			avail:  6,
			want:   [][]int{{0, 1, 2}, {3, 4, 5}},
		},
		{
			name:   "exact-fit item stays in the row",
			widths: []int{3, 3},
			gap:    0,
			avail:  6,
			want:   [][]int{{0, 1}},
		},
		{
			name:   "one unit over wraps",
			widths: []int{3, 4},
			gap:    0,
			avail:  6,
			want:   [][]int{{0}, {1}},
		},
		{
			name:   "single item exactly fills the row",
			widths: []int{6},
			gap:    0,
			avail:  6,
			want:   [][]int{{0}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := packRows(tc.widths, tc.gap, tc.avail)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("packRows(%v, %d, %d) = %v, want %v", tc.widths, tc.gap, tc.avail, got, tc.want)
			}
		})
	}
}

// TestPackRows_Boundaries references AC-2: the gap counts toward the used
// width when deciding whether the next item fits; an item wider than the
// available width always occupies its own row; avail <= 0 degrades to one
// item per row; empty input yields no rows.
func TestPackRows_Boundaries(t *testing.T) {
	cases := []struct {
		name   string
		widths []int
		gap    int
		avail  int
		want   [][]int
	}{
		{
			name:   "gap counted: fits without the gap but not with it, wraps",
			widths: []int{3, 3},
			gap:    1,
			avail:  6,
			want:   [][]int{{0}, {1}},
		},
		{
			name:   "gap counted: fits including the gap, stays",
			widths: []int{3, 2},
			gap:    1,
			avail:  6,
			want:   [][]int{{0, 1}},
		},
		{
			name:   "item wider than avail gets its own row, sandwiched between fitting items",
			widths: []int{1, 10, 1},
			gap:    0,
			avail:  6,
			want:   [][]int{{0}, {1}, {2}},
		},
		{
			name:   "avail == 0: one item per row",
			widths: []int{1, 2, 3},
			gap:    0,
			avail:  0,
			want:   [][]int{{0}, {1}, {2}},
		},
		{
			name:   "avail < 0: one item per row",
			widths: []int{1, 2, 3},
			gap:    0,
			avail:  -5,
			want:   [][]int{{0}, {1}, {2}},
		},
		{
			name:   "empty input: no rows",
			widths: nil,
			gap:    0,
			avail:  6,
			want:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := packRows(tc.widths, tc.gap, tc.avail)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("packRows(%v, %d, %d) = %v, want %v", tc.widths, tc.gap, tc.avail, got, tc.want)
			}
		})
	}
}

// --- packRowsWithBreaks (section-row-break task0001) ---

// TestPackRowsWithBreaks_MatchesPackRowsWhenNoBreaks references AC-5: with
// no forced breaks, packRowsWithBreaks must produce exactly the same
// assignment as packRows given the same widths, gap, and avail.
func TestPackRowsWithBreaks_MatchesPackRowsWhenNoBreaks(t *testing.T) {
	widths := []int{1, 2, 2, 2, 2, 2}
	gap := 0
	avail := 6

	want := packRows(widths, gap, avail)
	got := packRowsWithBreaks(widths, nil, gap, avail)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("packRowsWithBreaks(%v, nil, %d, %d) = %v, want %v (equal to packRows)", widths, gap, avail, got, want)
	}
}

// TestPackRowsWithBreaks_ForcesRowEndAtBreak references AC-1: a break
// forces the current row to end there even when the following item would
// otherwise fit in the same row.
func TestPackRowsWithBreaks_ForcesRowEndAtBreak(t *testing.T) {
	// Widths 1 and 2 easily fit together in avail=10 without a break.
	widths := []int{1, 2}
	gap := 0
	avail := 10

	withoutBreak := packRowsWithBreaks(widths, nil, gap, avail)
	if !reflect.DeepEqual(withoutBreak, [][]int{{0, 1}}) {
		t.Fatalf("packRowsWithBreaks(%v, nil, %d, %d) = %v, want both in one row (sanity check)", widths, gap, avail, withoutBreak)
	}

	withBreak := packRowsWithBreaks(widths, []int{1}, gap, avail)
	want := [][]int{{0}, {1}}
	if !reflect.DeepEqual(withBreak, want) {
		t.Errorf("packRowsWithBreaks(%v, []int{1}, %d, %d) = %v, want %v (break forces a new row)", widths, gap, avail, withBreak, want)
	}
}

// TestPackRowsWithBreaks_SegmentsPackIndependently references AC-5: each
// break-delimited segment is packed with the same greedy width algorithm as
// packRows, independently of the other segments.
func TestPackRowsWithBreaks_SegmentsPackIndependently(t *testing.T) {
	// Segment 1 (indices 0-2): widths 1,2,2 pack into one row of 5 (avail 6).
	// Segment 2 (indices 3-5): widths 2,2,2 also pack into one row of 6.
	// Without the break at index 3, all six would still pack as two rows of
	// three (matching TestPackRows' first case) -- this test forces the
	// same split via an explicit break instead of via width overflow.
	widths := []int{1, 2, 2, 2, 2, 2}
	gap := 0
	avail := 6

	got := packRowsWithBreaks(widths, []int{3}, gap, avail)
	want := [][]int{{0, 1, 2}, {3, 4, 5}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("packRowsWithBreaks(%v, []int{3}, %d, %d) = %v, want %v", widths, gap, avail, got, want)
	}
}

// TestPackRowsWithBreaks_EmptySegmentsProduceNoRows references AC-6:
// leading, trailing, and consecutive breaks -- and a breaks-only input --
// produce no empty rows.
func TestPackRowsWithBreaks_EmptySegmentsProduceNoRows(t *testing.T) {
	cases := []struct {
		name   string
		widths []int
		breaks []int
		want   [][]int
	}{
		{
			name:   "leading break at index 0",
			widths: []int{2, 2},
			breaks: []int{0},
			want:   [][]int{{0, 1}},
		},
		{
			name:   "trailing break at the end",
			widths: []int{2, 2},
			breaks: []int{2},
			want:   [][]int{{0, 1}},
		},
		{
			name:   "consecutive breaks with nothing between them",
			widths: []int{2, 2},
			breaks: []int{1, 1},
			want:   [][]int{{0}, {1}},
		},
		{
			name:   "breaks-only, zero real sections",
			widths: nil,
			breaks: []int{0, 0},
			want:   nil,
		},
		{
			name:   "zero entries, zero breaks",
			widths: nil,
			breaks: nil,
			want:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := packRowsWithBreaks(tc.widths, tc.breaks, 0, 6)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("packRowsWithBreaks(%v, %v, 0, 6) = %v, want %v", tc.widths, tc.breaks, got, tc.want)
			}
		})
	}
}

// TestPackRowsWithBreaks_DividerCountsMatchNaturalWrapping references AC-7:
// the divider counts derived from a break-produced assignment equal those
// of the same-shaped assignment produced by natural wrapping -- proving
// rowDividerCounts treats a forced break exactly like a natural wrap.
func TestPackRowsWithBreaks_DividerCountsMatchNaturalWrapping(t *testing.T) {
	// Forced break at index 3: segment [1,1,1] all fit one row (avail
	// 1000), segment [50,50] also fits one row on its own -- the break,
	// not width overflow, is what ends the first row.
	viaBreak := packRowsWithBreaks([]int{1, 1, 1, 50, 50}, []int{3}, 0, 1000)

	// Natural wrapping producing the identical assignment, with no break
	// involved at all: avail sized so packRows itself wraps after the
	// third item purely from width overflow.
	viaNaturalWrap := packRows([]int{1, 1, 1, 1, 1}, 0, 3)

	want := [][]int{{0, 1, 2}, {3, 4}}
	if !reflect.DeepEqual(viaBreak, want) {
		t.Fatalf("packRowsWithBreaks(...) = %v, want %v (sanity check for this test)", viaBreak, want)
	}
	if !reflect.DeepEqual(viaNaturalWrap, want) {
		t.Fatalf("packRows(...) = %v, want %v (sanity check for this test)", viaNaturalWrap, want)
	}

	gotH, gotV := rowDividerCounts(viaBreak)
	wantH, wantV := rowDividerCounts(viaNaturalWrap)
	if gotH != wantH || !reflect.DeepEqual(gotV, wantV) {
		t.Errorf("rowDividerCounts(viaBreak) = (%d, %v), want (%d, %v) (equal to natural wrapping)", gotH, gotV, wantH, wantV)
	}
}

// --- assignmentsEqual (task0001) ---

// TestAssignmentsEqual references AC-5: the equality helper reflow uses to
// decide whether to skip a row-widget rebuild.
func TestAssignmentsEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b [][]int
		want bool
	}{
		{
			name: "both nil (no sections)",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "identical shape and values",
			a:    [][]int{{0, 1}, {2}},
			b:    [][]int{{0, 1}, {2}},
			want: true,
		},
		{
			name: "different row count",
			a:    [][]int{{0, 1}},
			b:    [][]int{{0}, {1}},
			want: false,
		},
		{
			name: "different item count in a row",
			a:    [][]int{{0, 1}},
			b:    [][]int{{0, 1, 2}},
			want: false,
		},
		{
			name: "same items, different order",
			a:    [][]int{{0, 1}},
			b:    [][]int{{1, 0}},
			want: false,
		},
		{
			name: "different values at the same position",
			a:    [][]int{{0, 1}},
			b:    [][]int{{0, 2}},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := assignmentsEqual(tc.a, tc.b); got != tc.want {
				t.Errorf("assignmentsEqual(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// --- rowDividerCounts (task0001) ---

// TestRowDividerCounts references AC-5: after a rebuild with N rows,
// exactly N-1 horizontal dividers exist between the row boxes; a row
// holding M sections holds exactly M-1 vertical dividers; zero sections
// produce zero dividers (E-2); a single row produces zero horizontal
// dividers (E-1); a single-section row produces zero vertical dividers.
func TestRowDividerCounts(t *testing.T) {
	cases := []struct {
		name           string
		assignment     [][]int
		wantHorizontal int
		wantVertical   []int
	}{
		{
			name:           "zero sections: no rows, no dividers",
			assignment:     nil,
			wantHorizontal: 0,
			wantVertical:   []int{},
		},
		{
			name:           "single row, single section: no dividers at all",
			assignment:     [][]int{{0}},
			wantHorizontal: 0,
			wantVertical:   []int{0},
		},
		{
			name:           "single row, multiple sections: vertical dividers only",
			assignment:     [][]int{{0, 1, 2}},
			wantHorizontal: 0,
			wantVertical:   []int{2},
		},
		{
			name:           "multiple rows, single section each: horizontal dividers only",
			assignment:     [][]int{{0}, {1}, {2}},
			wantHorizontal: 2,
			wantVertical:   []int{0, 0, 0},
		},
		{
			name:           "multiple rows, mixed section counts",
			assignment:     [][]int{{0, 1}, {2}, {3, 4, 5}},
			wantHorizontal: 2,
			wantVertical:   []int{1, 0, 2},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotH, gotV := rowDividerCounts(tc.assignment)
			if gotH != tc.wantHorizontal {
				t.Errorf("rowDividerCounts(%v) horizontal = %d, want %d", tc.assignment, gotH, tc.wantHorizontal)
			}
			if !reflect.DeepEqual(gotV, tc.wantVertical) {
				t.Errorf("rowDividerCounts(%v) vertical = %v, want %v", tc.assignment, gotV, tc.wantVertical)
			}
		})
	}
}
