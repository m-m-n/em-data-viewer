package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ResolvePath decides the config file path: an explicit override
// (e.g. a CLI argument) wins if non-empty; otherwise
// $XDG_CONFIG_HOME/data-viewer/config.yaml if xdgConfigHome is
// non-empty; otherwise ~/.config/data-viewer/config.yaml using home.
// It is a pure function so path resolution is testable without
// touching the real environment or filesystem.
func ResolvePath(explicit, xdgConfigHome, home string) string {
	if explicit != "" {
		return explicit
	}
	if xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "data-viewer", "config.yaml")
	}
	return filepath.Join(home, ".config", "data-viewer", "config.yaml")
}

// DefaultPath resolves the config path using the current process's
// real environment and home directory, honoring an explicit override
// if provided.
func DefaultPath(explicit string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve home directory: %w", err)
	}
	return ResolvePath(explicit, os.Getenv("XDG_CONFIG_HOME"), home), nil
}

// yamlConfig, yamlWindow, and yamlSection mirror the on-disk YAML
// schema. Optional numeric fields use pointers so Load can
// distinguish "omitted" (apply default) from an explicit value
// (including an explicit zero, which is a validation error for
// max_lines).
type yamlConfig struct {
	Window   yamlWindow    `yaml:"window"`
	Sections []yamlSection `yaml:"sections"`
}

// yamlWindow deliberately has no Height field (task0001): the window's
// height is derived from content at runtime, not configured. Because YAML
// decoding is non-strict, a `height` key present in a user's file is
// silently ignored -- no error, no warning.
type yamlWindow struct {
	Width     *int              `yaml:"width"`
	Style     map[string]string `yaml:"style"`
	KeepAbove *bool             `yaml:"keep_above"`
}

type yamlSection struct {
	Name     string            `yaml:"name"`
	Command  string            `yaml:"command"`
	Mode     string            `yaml:"mode"`
	Interval string            `yaml:"interval"`
	MaxLines *int              `yaml:"max_lines"`
	Style    map[string]string `yaml:"style"`
}

// Load reads, parses, and validates the config file at path. On
// success it returns a Config with defaults applied (window width
// 400, max_lines 100, keep_above true where omitted). On failure —
// unreadable file, malformed YAML, or a validation failure — it
// returns a zero Config and a descriptive error. Callers must not
// treat a failed Load's zero Config as a valid config; see Holder
// for last-good retention across failed reloads.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("config: read %s: %w", path, err)
	}

	var raw yamlConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
	}

	cfg := Config{
		Window: Window{
			Width:     DefaultWindowWidth,
			Style:     raw.Window.Style,
			KeepAbove: DefaultKeepAbove,
		},
	}
	if raw.Window.Width != nil {
		cfg.Window.Width = *raw.Window.Width
	}
	if raw.Window.KeepAbove != nil {
		cfg.Window.KeepAbove = *raw.Window.KeepAbove
	}

	cfg.Sections = make([]SectionConfig, 0, len(raw.Sections))
	for i, rs := range raw.Sections {
		sec, err := buildSection(rs)
		if err != nil {
			return Config{}, fmt.Errorf("config: section %d (%q): %w", i, rs.Name, err)
		}
		cfg.Sections = append(cfg.Sections, sec)
	}

	if err := checkDuplicateKeys(cfg.Sections); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// checkDuplicateKeys rejects a config whose sections are not uniquely
// identified by SectionConfig.Key(): downstream components (runner,
// ui, app) assume key uniqueness, and two sections with an identical
// key would otherwise silently collide (a leaked widget, a section
// whose process the runner cannot tell apart from another).
func checkDuplicateKeys(sections []SectionConfig) error {
	firstIndex := make(map[string]int, len(sections))
	for i, sec := range sections {
		key := sec.Key()
		if first, ok := firstIndex[key]; ok {
			return fmt.Errorf("config: sections %d and %d (name=%q command=%q mode=%q) have identical settings: duplicate section keys are not allowed", first, i, sec.Name, sec.Command, sec.Mode)
		}
		firstIndex[key] = i
	}
	return nil
}

// buildSection validates one raw YAML section and converts it into
// a SectionConfig, applying the max_lines default when omitted.
func buildSection(rs yamlSection) (SectionConfig, error) {
	if rs.Command == "" {
		return SectionConfig{}, fmt.Errorf("missing command")
	}

	switch rs.Mode {
	case ModeInterval, ModeStream:
	default:
		return SectionConfig{}, fmt.Errorf("unknown mode %q", rs.Mode)
	}

	sec := SectionConfig{
		Name:    rs.Name,
		Command: rs.Command,
		Mode:    rs.Mode,
		Style:   rs.Style,
	}

	if rs.Mode == ModeInterval {
		if rs.Interval == "" {
			return SectionConfig{}, fmt.Errorf("missing interval for interval mode")
		}
		d, err := time.ParseDuration(rs.Interval)
		if err != nil {
			return SectionConfig{}, fmt.Errorf("unparsable interval %q: %w", rs.Interval, err)
		}
		if d <= 0 {
			return SectionConfig{}, fmt.Errorf("non-positive interval %q", rs.Interval)
		}
		sec.Interval = d
	}

	sec.MaxLines = DefaultMaxLines
	if rs.MaxLines != nil {
		if *rs.MaxLines <= 0 {
			return SectionConfig{}, fmt.Errorf("non-positive max_lines %d", *rs.MaxLines)
		}
		sec.MaxLines = *rs.MaxLines
	}

	return sec, nil
}
