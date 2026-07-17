package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	testDebounce = 50 * time.Millisecond
	testTimeout  = 2 * time.Second
)

func waitNotify(t *testing.T, w *Watcher, timeout time.Duration) bool {
	t.Helper()
	select {
	case <-w.Notify():
		return true
	case <-time.After(timeout):
		return false
	}
}

// TestWatcher_NotifiesOnInPlaceWrite references AC-6 (in-place write).
func TestWatcher_NotifiesOnInPlaceWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("sections: []\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	w, err := NewWatcher(path, testDebounce)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	if err := os.WriteFile(path, []byte("sections: []\nwindow:\n  width: 500\n"), 0o644); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	if !waitNotify(t, w, testTimeout) {
		t.Fatal("watcher did not notify after in-place write")
	}
}

// TestWatcher_NotifiesOnRenameReplace references AC-6 (editor
// rename/replace save pattern: write to a temp file, then rename it
// over the target).
func TestWatcher_NotifiesOnRenameReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("sections: []\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	w, err := NewWatcher(path, testDebounce)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	tmp := filepath.Join(dir, "config.yaml.tmp")
	if err := os.WriteFile(tmp, []byte("sections: []\nwindow:\n  width: 700\n"), 0o644); err != nil {
		t.Fatalf("write tmp config: %v", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatalf("rename tmp over config: %v", err)
	}

	if !waitNotify(t, w, testTimeout) {
		t.Fatal("watcher did not notify after rename/replace save")
	}
}

// TestWatcher_DebouncesBurstIntoOneNotification references AC-6
// (debounce).
func TestWatcher_DebouncesBurstIntoOneNotification(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("sections: []\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	debounce := 200 * time.Millisecond
	w, err := NewWatcher(path, debounce)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	for i := 0; i < 5; i++ {
		if err := os.WriteFile(path, []byte("sections: []\n# burst\n"), 0o644); err != nil {
			t.Fatalf("burst write %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !waitNotify(t, w, testTimeout) {
		t.Fatal("watcher did not notify after write burst")
	}

	select {
	case <-w.Notify():
		t.Fatal("watcher sent a second notification for a single debounced burst")
	case <-time.After(150 * time.Millisecond):
	}
}

// TestWatcher_IgnoresUnrelatedFilesInSameDirectory guards the
// filename filter that lets the watcher watch the directory without
// firing on unrelated sibling files.
func TestWatcher_IgnoresUnrelatedFilesInSameDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("sections: []\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	w, err := NewWatcher(path, testDebounce)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	other := filepath.Join(dir, "unrelated.txt")
	if err := os.WriteFile(other, []byte("noise"), 0o644); err != nil {
		t.Fatalf("write unrelated file: %v", err)
	}

	if waitNotify(t, w, 300*time.Millisecond) {
		t.Fatal("watcher notified for a change to an unrelated file")
	}
}

// TestWatcher_SurvivesFileDeletion covers the edge case: the config
// file is deleted while running; the watcher keeps watching the
// directory (no crash) and still notifies once the file reappears.
func TestWatcher_SurvivesFileDeletion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("sections: []\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	w, err := NewWatcher(path, testDebounce)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove config: %v", err)
	}

	if err := os.WriteFile(path, []byte("sections: []\nwindow:\n  width: 321\n"), 0o644); err != nil {
		t.Fatalf("recreate config: %v", err)
	}

	if !waitNotify(t, w, testTimeout) {
		t.Fatal("watcher did not notify after file recreated post-deletion")
	}
}

// TestWatcher_NotifiesOnDelete references AC-1: deleting the watched
// config file (with no replacement) still produces a notification, so
// the app layer can react (last-good config + banner) instead of
// silently going stale.
func TestWatcher_NotifiesOnDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("sections: []\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	w, err := NewWatcher(path, testDebounce)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove config: %v", err)
	}

	if !waitNotify(t, w, testTimeout) {
		t.Fatal("watcher did not notify after the config file was deleted")
	}
}

// TestWatcher_NotifiesOnRenameAway references AC-1: renaming the
// watched config file to a different name (without replacing it) also
// produces a notification.
func TestWatcher_NotifiesOnRenameAway(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("sections: []\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	w, err := NewWatcher(path, testDebounce)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	defer w.Close()

	renamed := filepath.Join(dir, "config.yaml.bak")
	if err := os.Rename(path, renamed); err != nil {
		t.Fatalf("rename config away: %v", err)
	}

	if !waitNotify(t, w, testTimeout) {
		t.Fatal("watcher did not notify after the config file was renamed away")
	}
}

func TestWatcher_CloseStopsNotifications(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("sections: []\n"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	w, err := NewWatcher(path, testDebounce)
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if err := os.WriteFile(path, []byte("sections: []\n# after close\n"), 0o644); err != nil {
		t.Fatalf("write after close: %v", err)
	}

	if waitNotify(t, w, 300*time.Millisecond) {
		t.Fatal("watcher notified after Close()")
	}
}
