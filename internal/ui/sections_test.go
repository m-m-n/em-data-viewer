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
