package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUpdateKeepAbove_PreservesCommentsAndOtherKeys references AC-3:
// flipping keep_above must leave every other key, value, and comment
// byte-identical, changing only the keep_above line/value. It also
// covers creating the key when window: is present but keep_above is
// absent, and creating window: itself when absent.
func TestUpdateKeepAbove_PreservesCommentsAndOtherKeys(t *testing.T) {
	tests := []struct {
		name  string
		input string
		value bool
		want  string
	}{
		{
			name: "flips existing false to true, preserving comments",
			input: `# user config
window:
  width: 800 # custom width
  # style comment
  style:
    background: "#123456"
  keep_above: false
sections:
  - name: clock # inline comment
    command: date
    mode: interval
    interval: 1s
`,
			value: true,
			want: `# user config
window:
  width: 800 # custom width
  # style comment
  style:
    background: "#123456"
  keep_above: true
sections:
  - name: clock # inline comment
    command: date
    mode: interval
    interval: 1s
`,
		},
		{
			name: "flips existing true to false",
			input: `window:
  keep_above: true
sections: []
`,
			value: false,
			want: `window:
  keep_above: false
sections: []
`,
		},
		{
			name: "creates keep_above under existing window mapping",
			input: `# top comment
window:
  width: 800
sections: []
`,
			value: false,
			want: `# top comment
window:
  width: 800
  keep_above: false
sections: []
`,
		},
		{
			name: "creates window mapping when absent",
			input: `# top comment
sections:
  - name: clock
    command: date
    mode: interval
    interval: 1s
`,
			value: false,
			want: `# top comment
sections:
  - name: clock
    command: date
    mode: interval
    interval: 1s
window:
  keep_above: false
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempConfig(t, tt.input)

			if err := UpdateKeepAbove(path, tt.value); err != nil {
				t.Fatalf("UpdateKeepAbove() error = %v, want nil", err)
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read result: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("UpdateKeepAbove() result =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

// TestUpdateKeepAbove_ErrorsLeaveFileUnchanged references AC-4:
// a missing, unparsable, or unwritable target returns an error and
// leaves the original file (if any) byte-identical to before the
// call.
func TestUpdateKeepAbove_ErrorsLeaveFileUnchanged(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "does-not-exist.yaml")

		if err := UpdateKeepAbove(path, true); err == nil {
			t.Fatal("UpdateKeepAbove() error = nil, want error for missing file")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("target file should still not exist, stat err = %v", err)
		}
	})

	t.Run("unparsable yaml", func(t *testing.T) {
		original := "sections: [this is not: valid: yaml"
		path := writeTempConfig(t, original)

		if err := UpdateKeepAbove(path, true); err == nil {
			t.Fatal("UpdateKeepAbove() error = nil, want error for unparsable yaml")
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read result: %v", err)
		}
		if string(got) != original {
			t.Errorf("file changed on parse failure: got %q, want unchanged %q", got, original)
		}
	})

	t.Run("unwritable directory", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		original := "window:\n  keep_above: false\nsections: []\n"
		if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
			t.Fatalf("write temp config: %v", err)
		}
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatalf("chmod dir read-only: %v", err)
		}
		t.Cleanup(func() {
			_ = os.Chmod(dir, 0o755)
		})

		if err := UpdateKeepAbove(path, true); err == nil {
			t.Fatal("UpdateKeepAbove() error = nil, want error for unwritable directory")
		}

		if err := os.Chmod(dir, 0o755); err != nil {
			t.Fatalf("restore dir permissions: %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read result: %v", err)
		}
		if string(got) != original {
			t.Errorf("file changed despite unwritable directory: got %q, want unchanged %q", got, original)
		}
	})

	t.Run("top-level yaml is not a mapping", func(t *testing.T) {
		original := "- just\n- a\n- list\n"
		path := writeTempConfig(t, original)

		if err := UpdateKeepAbove(path, true); err == nil {
			t.Fatal("UpdateKeepAbove() error = nil, want error for non-mapping top level")
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read result: %v", err)
		}
		if string(got) != original {
			t.Errorf("file changed on non-mapping error: got %q, want unchanged %q", got, original)
		}
	})
}

// TestUpdateKeepAbove_AtomicWriteLeavesNoTempFileBehind references
// AC-5: the write is temp-file + rename, so after a successful call
// only the target path exists in the directory (no stray temp file).
func TestUpdateKeepAbove_AtomicWriteLeavesNoTempFileBehind(t *testing.T) {
	path := writeTempConfig(t, "window:\n  keep_above: false\nsections: []\n")
	dir := filepath.Dir(path)

	if err := UpdateKeepAbove(path, true); err != nil {
		t.Fatalf("UpdateKeepAbove() error = %v, want nil", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("directory entries = %v, want only %q (no stray temp file)", names, filepath.Base(path))
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read result: %v", err)
	}
	if !strings.Contains(string(got), "keep_above: true") {
		t.Errorf("result = %q, want it to contain keep_above: true", got)
	}
}
