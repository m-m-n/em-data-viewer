package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

// TestLoad_ValidConfigAppliesDefaults references AC-1.
func TestLoad_ValidConfigAppliesDefaults(t *testing.T) {
	t.Run("defaults applied when window and max_lines omitted", func(t *testing.T) {
		path := writeTempConfig(t, `
sections:
  - name: clock
    command: "date"
    mode: interval
    interval: 1s
`)
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if cfg.Window.Width != DefaultWindowWidth {
			t.Errorf("Window.Width = %d, want default %d", cfg.Window.Width, DefaultWindowWidth)
		}
		if len(cfg.Sections) != 1 {
			t.Fatalf("len(Sections) = %d, want 1", len(cfg.Sections))
		}
		sec := cfg.Sections[0]
		if sec.MaxLines != DefaultMaxLines {
			t.Errorf("MaxLines = %d, want default %d", sec.MaxLines, DefaultMaxLines)
		}
		if sec.Command != "date" || sec.Mode != ModeInterval || sec.Interval != time.Second {
			t.Errorf("section = %+v, unexpected values", sec)
		}
		if sec.Name != "clock" {
			t.Errorf("Name = %q, want %q", sec.Name, "clock")
		}
	})

	t.Run("explicit window width and max_lines override defaults", func(t *testing.T) {
		path := writeTempConfig(t, `
window:
  width: 800
sections:
  - command: "tail -f /tmp/foo"
    mode: stream
    max_lines: 50
`)
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if cfg.Window.Width != 800 {
			t.Errorf("Window.Width = %d, want 800", cfg.Window.Width)
		}
		if cfg.Sections[0].MaxLines != 50 {
			t.Errorf("MaxLines = %d, want 50", cfg.Sections[0].MaxLines)
		}
	})

	// TestLoad_ValidConfigAppliesDefaults/window.height_key_is_silently_ignored
	// references AC-1: a `window.height` key in YAML loads with no error and
	// has no effect on the loaded config (the field no longer exists on
	// Window, so there is nothing it could set).
	t.Run("window.height key is silently ignored", func(t *testing.T) {
		path := writeTempConfig(t, `
window:
  width: 800
  height: 999
sections: []
`)
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v, want nil (a window.height key must not cause an error)", err)
		}
		if cfg.Window.Width != 800 {
			t.Errorf("Window.Width = %d, want 800 (an unrelated window.height key must not affect other fields)", cfg.Window.Width)
		}
	})

	t.Run("zero sections is valid", func(t *testing.T) {
		path := writeTempConfig(t, `window:
  width: 400
`)
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if len(cfg.Sections) != 0 {
			t.Errorf("len(Sections) = %d, want 0", len(cfg.Sections))
		}
	})

	t.Run("window style values are preserved", func(t *testing.T) {
		path := writeTempConfig(t, `
window:
  style:
    background: "#1e1e1eee"
sections: []
`)
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if got := cfg.Window.Style["background"]; got != "#1e1e1eee" {
			t.Errorf("Window.Style[background] = %q, want %q", got, "#1e1e1eee")
		}
	})
}

// TestLoad_KeepAboveDefaultAndOverride references AC-1: keep_above
// absent defaults to true; an explicit false or true is honored.
func TestLoad_KeepAboveDefaultAndOverride(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want bool
	}{
		{
			name: "absent defaults to true",
			yaml: `sections: []`,
			want: true,
		},
		{
			name: "explicit false",
			yaml: `
window:
  keep_above: false
sections: []
`,
			want: false,
		},
		{
			name: "explicit true",
			yaml: `
window:
  keep_above: true
sections: []
`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempConfig(t, tt.yaml)
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v, want nil", err)
			}
			if cfg.Window.KeepAbove != tt.want {
				t.Errorf("Window.KeepAbove = %v, want %v", cfg.Window.KeepAbove, tt.want)
			}
		})
	}
}

// TestLoad_ValidationErrors references AC-2.
func TestLoad_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "missing command",
			yaml: `
sections:
  - mode: interval
    interval: 1s
`,
		},
		{
			name: "unknown mode",
			yaml: `
sections:
  - command: "date"
    mode: bogus
`,
		},
		{
			name: "missing interval for interval mode",
			yaml: `
sections:
  - command: "date"
    mode: interval
`,
		},
		{
			name: "unparsable interval for interval mode",
			yaml: `
sections:
  - command: "date"
    mode: interval
    interval: "not-a-duration"
`,
		},
		{
			name: "zero max_lines",
			yaml: `
sections:
  - command: "tail -f /tmp/foo"
    mode: stream
    max_lines: 0
`,
		},
		{
			name: "negative max_lines",
			yaml: `
sections:
  - command: "tail -f /tmp/foo"
    mode: stream
    max_lines: -3
`,
		},
		{
			name: "malformed yaml",
			yaml: "sections: [this is not: valid: yaml",
		},
		{
			name: "zero interval for interval mode",
			yaml: `
sections:
  - command: "date"
    mode: interval
    interval: 0s
`,
		},
		{
			name: "negative interval for interval mode",
			yaml: `
sections:
  - command: "date"
    mode: interval
    interval: -1s
`,
		},
		{
			name: "non-boolean keep_above",
			yaml: `
window:
  keep_above: "not-a-bool"
sections: []
`,
		},
		{
			name: "duplicate section keys",
			yaml: `
sections:
  - name: clock
    command: "date"
    mode: interval
    interval: 1s
  - name: clock
    command: "date"
    mode: interval
    interval: 1s
`,
		},
		{
			name: "unknown type",
			yaml: `
sections:
  - type: bogus
    command: "date"
    mode: interval
    interval: 1s
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempConfig(t, tt.yaml)
			cfg, err := Load(path)
			if err == nil {
				t.Fatalf("Load() error = nil, want descriptive error")
			}
			if err.Error() == "" {
				t.Errorf("Load() error message is empty")
			}
			if !reflect.DeepEqual(cfg, Config{}) {
				t.Errorf("Load() config = %+v, want zero value on error", cfg)
			}
		})
	}
}

// TestLoad_DuplicateSectionKeysNamesOffendingIndices references AC-2:
// the duplicate-key validation error names the offending section
// indices so the user can find them in their config file.
func TestLoad_DuplicateSectionKeysNamesOffendingIndices(t *testing.T) {
	path := writeTempConfig(t, `
sections:
  - name: clock
    command: "date"
    mode: interval
    interval: 1s
  - name: uptime
    command: "uptime"
    mode: interval
    interval: 1s
  - name: clock
    command: "date"
    mode: interval
    interval: 1s
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want duplicate-key error")
	}
	if !strings.Contains(err.Error(), "0") || !strings.Contains(err.Error(), "2") {
		t.Errorf("Load() error = %q, want it to name offending indices 0 and 2", err.Error())
	}
}

// --- section-row-break task0001: break entries (TS-1..TS-4) ---

// TestLoad_BreakEntryLoadsSuccessfully references AC-1: a `- type: break`
// entry between two sections loads successfully, preserving section order,
// and is marked as a break so downstream layers can recognize and skip it.
func TestLoad_BreakEntryLoadsSuccessfully(t *testing.T) {
	path := writeTempConfig(t, `
sections:
  - name: clock
    command: "date"
    mode: interval
    interval: 1s
  - type: break
  - name: uptime
    command: "uptime"
    mode: interval
    interval: 1s
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}
	if len(cfg.Sections) != 3 {
		t.Fatalf("len(Sections) = %d, want 3", len(cfg.Sections))
	}
	if cfg.Sections[0].IsBreak {
		t.Errorf("Sections[0].IsBreak = true, want false (clock)")
	}
	if !cfg.Sections[1].IsBreak {
		t.Errorf("Sections[1].IsBreak = false, want true (break entry)")
	}
	if cfg.Sections[2].IsBreak {
		t.Errorf("Sections[2].IsBreak = true, want false (uptime)")
	}
	if cfg.Sections[0].Name != "clock" || cfg.Sections[2].Name != "uptime" {
		t.Errorf("Sections order not preserved around the break: %+v", cfg.Sections)
	}
}

// TestLoad_BreakTypeIgnoresOtherFields references AC-2: type: break with
// other fields present still loads as a break; the other fields are not
// validated and do not appear on the resulting SectionConfig.
func TestLoad_BreakTypeIgnoresOtherFields(t *testing.T) {
	path := writeTempConfig(t, `
sections:
  - type: break
    name: ignored
    command: "should not matter"
    mode: bogus-mode
    interval: "not-a-duration"
    max_lines: -1
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil (a break entry skips validation of the other fields)", err)
	}
	if len(cfg.Sections) != 1 {
		t.Fatalf("len(Sections) = %d, want 1", len(cfg.Sections))
	}
	if !reflect.DeepEqual(cfg.Sections[0], SectionConfig{IsBreak: true}) {
		t.Errorf("Sections[0] = %+v, want a bare break marker with all other fields zero-valued", cfg.Sections[0])
	}
}

// TestLoad_UnknownTypeFailsValidation references AC-2: type set to any
// value other than "break" (or omitted) fails validation, naming the
// offending value.
func TestLoad_UnknownTypeFailsValidation(t *testing.T) {
	path := writeTempConfig(t, `
sections:
  - type: bogus
    command: "date"
    mode: interval
    interval: 1s
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want validation error for unknown type")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("Load() error = %q, want it to name the offending type value %q", err.Error(), "bogus")
	}
}

// TestLoad_MultipleBreaksNoDuplicateKeyError references AC-3: two or more
// break entries in one config do not trigger the duplicate-key error.
func TestLoad_MultipleBreaksNoDuplicateKeyError(t *testing.T) {
	path := writeTempConfig(t, `
sections:
  - type: break
  - type: break
  - name: clock
    command: "date"
    mode: interval
    interval: 1s
  - type: break
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil (multiple breaks must not trigger the duplicate-key error)", err)
	}
	if len(cfg.Sections) != 4 {
		t.Fatalf("len(Sections) = %d, want 4", len(cfg.Sections))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(filepath.Join(dir, "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("Load() error = nil, want error for missing file")
	}
}

// TestResolvePath references AC-3.
func TestResolvePath(t *testing.T) {
	tests := []struct {
		name          string
		explicit      string
		xdgConfigHome string
		home          string
		want          string
	}{
		{
			name:          "explicit path wins over xdg and home",
			explicit:      "/custom/config.yaml",
			xdgConfigHome: "/xdg",
			home:          "/home/user",
			want:          "/custom/config.yaml",
		},
		{
			name:          "xdg config home used when no explicit path",
			xdgConfigHome: "/xdg",
			home:          "/home/user",
			want:          filepath.Join("/xdg", "em-data-viewer", "config.yaml"),
		},
		{
			name: "home fallback when neither explicit nor xdg set",
			home: "/home/user",
			want: filepath.Join("/home/user", ".config", "em-data-viewer", "config.yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolvePath(tt.explicit, tt.xdgConfigHome, tt.home)
			if got != tt.want {
				t.Errorf("ResolvePath(%q, %q, %q) = %q, want %q", tt.explicit, tt.xdgConfigHome, tt.home, got, tt.want)
			}
		})
	}
}

// TestSectionConfig_Key references AC-4.
func TestSectionConfig_Key(t *testing.T) {
	base := SectionConfig{
		Command:  "date",
		Mode:     ModeInterval,
		Interval: time.Second,
		MaxLines: DefaultMaxLines,
		Style:    map[string]string{"color": "#ffffff"},
	}

	t.Run("stable under style-only changes", func(t *testing.T) {
		styled := base
		styled.Style = map[string]string{"color": "#000000", "font": "monospace 12"}
		if base.Key() != styled.Key() {
			t.Errorf("Key() differs on style-only change: %q vs %q", base.Key(), styled.Key())
		}
	})

	t.Run("differs when command changes", func(t *testing.T) {
		other := base
		other.Command = "uptime"
		if base.Key() == other.Key() {
			t.Errorf("Key() unchanged when Command changed: %q", base.Key())
		}
	})

	t.Run("differs when mode changes", func(t *testing.T) {
		other := base
		other.Mode = ModeStream
		if base.Key() == other.Key() {
			t.Errorf("Key() unchanged when Mode changed: %q", base.Key())
		}
	})

	t.Run("differs when interval changes", func(t *testing.T) {
		other := base
		other.Interval = 2 * time.Second
		if base.Key() == other.Key() {
			t.Errorf("Key() unchanged when Interval changed: %q", base.Key())
		}
	})

	t.Run("differs when max_lines changes", func(t *testing.T) {
		other := base
		other.MaxLines = 50
		if base.Key() == other.Key() {
			t.Errorf("Key() unchanged when MaxLines changed: %q", base.Key())
		}
	})
}
