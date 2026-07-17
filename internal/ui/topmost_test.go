package ui

import "testing"

// TestDeriveKeepAboveState references AC-1: the pure state-derivation
// function returns (check, sensitive, hint) correctly for all (enabled,
// supported) combinations.
func TestDeriveKeepAboveState(t *testing.T) {
	cases := []struct {
		name      string
		enabled   bool
		supported bool
		want      keepAboveState
	}{
		{
			name:      "enabled and supported: checked, sensitive, no hint",
			enabled:   true,
			supported: true,
			want:      keepAboveState{Checked: true, Sensitive: true, Hint: ""},
		},
		{
			name:      "disabled but supported: unchecked, sensitive, no hint",
			enabled:   false,
			supported: true,
			want:      keepAboveState{Checked: false, Sensitive: true, Hint: ""},
		},
		{
			name:      "enabled but unsupported: still checked, insensitive, hint shown (AC-4, AC-5)",
			enabled:   true,
			supported: false,
			want:      keepAboveState{Checked: true, Sensitive: false, Hint: keepAboveWaylandHint},
		},
		{
			name:      "disabled and unsupported: unchecked, insensitive, hint shown",
			enabled:   false,
			supported: false,
			want:      keepAboveState{Checked: false, Sensitive: false, Hint: keepAboveWaylandHint},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveKeepAboveState(tc.enabled, tc.supported)
			if got != tc.want {
				t.Errorf("deriveKeepAboveState(%v, %v) = %+v, want %+v", tc.enabled, tc.supported, got, tc.want)
			}
		})
	}
}

// TestDeriveKeepAboveState_HintOnlyWhenUnsupported guards against a hint
// leaking into the supported case regardless of the enabled value (AC-1).
func TestDeriveKeepAboveState_HintOnlyWhenUnsupported(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		if got := deriveKeepAboveState(enabled, true).Hint; got != "" {
			t.Errorf("deriveKeepAboveState(%v, true).Hint = %q, want empty", enabled, got)
		}
		if got := deriveKeepAboveState(enabled, false).Hint; got != keepAboveWaylandHint {
			t.Errorf("deriveKeepAboveState(%v, false).Hint = %q, want %q", enabled, got, keepAboveWaylandHint)
		}
	}
}

// TestSelectKeepAboveBackend_NilDisplayIsInert guards the degrade-safely
// path: an unclassifiable (nil) display must never select the X11 backend
// (SPEC FR4 "no X11 calls are made" on non-X11 backends).
func TestSelectKeepAboveBackend_NilDisplayIsInert(t *testing.T) {
	backend := selectKeepAboveBackend(nil)
	if _, ok := backend.(inertKeepAboveBackend); !ok {
		t.Errorf("selectKeepAboveBackend(nil) = %T, want inertKeepAboveBackend", backend)
	}
	if backend.Supported(nil) {
		t.Error("inertKeepAboveBackend.Supported(nil) = true, want false")
	}
	if err := backend.Apply(nil, true); err != nil {
		t.Errorf("inertKeepAboveBackend.Apply(nil, true) = %v, want nil (no-op)", err)
	}
}
