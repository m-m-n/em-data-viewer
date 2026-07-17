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
		if cfg.Window.Width != DefaultWindowWidth || cfg.Window.Height != DefaultWindowHeight {
			t.Errorf("Window = %+v, want defaults %dx%d", cfg.Window, DefaultWindowWidth, DefaultWindowHeight)
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

	t.Run("explicit window and max_lines override defaults", func(t *testing.T) {
		path := writeTempConfig(t, `
window:
  width: 800
  height: 200
sections:
  - command: "tail -f /tmp/foo"
    mode: stream
    max_lines: 50
`)
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v, want nil", err)
		}
		if cfg.Window.Width != 800 || cfg.Window.Height != 200 {
			t.Errorf("Window = %+v, want 800x200", cfg.Window)
		}
		if cfg.Sections[0].MaxLines != 50 {
			t.Errorf("MaxLines = %d, want 50", cfg.Sections[0].MaxLines)
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
			want:          filepath.Join("/xdg", "data-viewer", "config.yaml"),
		},
		{
			name: "home fallback when neither explicit nor xdg set",
			home: "/home/user",
			want: filepath.Join("/home/user", ".config", "data-viewer", "config.yaml"),
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
