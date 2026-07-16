package app

import (
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"data-viewer/internal/config"
	"data-viewer/internal/runner"
	"data-viewer/internal/ui"
)

// --- fakes (no GTK; UI/Runner/Watcher are app-boundary interfaces) ---

type sectionUpdate struct {
	key     string
	markup  string
	isError bool
}

type fakeUI struct {
	windowSettings []ui.WindowSettings
	sections       [][]ui.SectionViewModel
	updates        []sectionUpdate
	configErrors   []string
}

func (f *fakeUI) ApplyWindow(settings ui.WindowSettings) {
	f.windowSettings = append(f.windowSettings, settings)
}
func (f *fakeUI) SetSections(sections []ui.SectionViewModel) {
	f.sections = append(f.sections, sections)
}
func (f *fakeUI) UpdateSection(key, markup string, isError bool) {
	f.updates = append(f.updates, sectionUpdate{key, markup, isError})
}
func (f *fakeUI) SetConfigError(message string) {
	f.configErrors = append(f.configErrors, message)
}
func (f *fakeUI) Run() int { return 0 }

type fakeRunner struct {
	applyCalls [][]runner.SectionConfig
	stopCalls  int
	snapshots  map[string][]string
}

func (f *fakeRunner) Apply(sections []runner.SectionConfig) {
	f.applyCalls = append(f.applyCalls, sections)
}
func (f *fakeRunner) Stop()                        { f.stopCalls++ }
func (f *fakeRunner) Events() <-chan runner.Event  { return nil }
func (f *fakeRunner) Snapshot(key string) []string { return f.snapshots[key] }

// --- AC-1: config path resolution ---

// TestResolveConfigPath references AC-1.
func TestResolveConfigPath(t *testing.T) {
	t.Run("explicit argument overrides the XDG path", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/xdg-home")
		got, err := ResolveConfigPath("/custom/config.yaml")
		if err != nil {
			t.Fatalf("ResolveConfigPath() error = %v, want nil", err)
		}
		if got != "/custom/config.yaml" {
			t.Errorf("ResolveConfigPath() = %q, want %q", got, "/custom/config.yaml")
		}
	})

	t.Run("no argument resolves under XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/xdg-home")
		got, err := ResolveConfigPath("")
		if err != nil {
			t.Fatalf("ResolveConfigPath() error = %v, want nil", err)
		}
		want := filepath.Join("/xdg-home", "data-viewer", "config.yaml")
		if got != want {
			t.Errorf("ResolveConfigPath() = %q, want %q", got, want)
		}
	})
}

// --- AC-2, AC-3: reload / applyConfig decision logic ---

func sampleConfig() config.Config {
	return config.Config{
		Window: config.Window{Width: 800, Height: 200, Style: map[string]string{"background": "#000"}},
		Sections: []config.SectionConfig{
			{Name: "clock", Command: "date", Mode: config.ModeInterval, Interval: 5 * time.Second, MaxLines: 100},
			{Name: "", Command: "tail -f /tmp/x", Mode: config.ModeStream, MaxLines: 50},
		},
	}
}

// TestApplyConfig_SuccessAppliesWindowSectionsAndRunner references AC-2 and
// AC-3 (successful reload).
func TestApplyConfig_SuccessAppliesWindowSectionsAndRunner(t *testing.T) {
	cfg := sampleConfig()
	u := &fakeUI{}
	r := &fakeRunner{}

	applyConfig(cfg, nil, u, r)

	if len(u.configErrors) != 1 || u.configErrors[0] != "" {
		t.Fatalf("configErrors = %v, want a single cleared banner", u.configErrors)
	}
	if len(u.windowSettings) != 1 {
		t.Fatalf("windowSettings calls = %d, want 1", len(u.windowSettings))
	}
	wantWindow := ui.WindowSettings{Width: 800, Height: 200, Style: cfg.Window.Style}
	if !reflect.DeepEqual(u.windowSettings[0], wantWindow) {
		t.Errorf("ApplyWindow got %+v, want %+v", u.windowSettings[0], wantWindow)
	}

	if len(u.sections) != 1 {
		t.Fatalf("SetSections calls = %d, want 1", len(u.sections))
	}
	wantSections := []ui.SectionViewModel{
		{Key: cfg.Sections[0].Key(), Label: "clock", Style: cfg.Sections[0].Style},
		{Key: cfg.Sections[1].Key(), Label: "", Style: cfg.Sections[1].Style},
	}
	if !reflect.DeepEqual(u.sections[0], wantSections) {
		t.Errorf("SetSections got %+v, want %+v", u.sections[0], wantSections)
	}

	if len(r.applyCalls) != 1 {
		t.Fatalf("runner.Apply calls = %d, want 1", len(r.applyCalls))
	}
	wantRunnerSections := []runner.SectionConfig{
		{Key: cfg.Sections[0].Key(), Command: "date", Mode: runner.ModeInterval, Interval: 5 * time.Second, MaxLines: 100},
		{Key: cfg.Sections[1].Key(), Command: "tail -f /tmp/x", Mode: runner.ModeStream, MaxLines: 50},
	}
	if !reflect.DeepEqual(r.applyCalls[0], wantRunnerSections) {
		t.Errorf("runner.Apply got %+v, want %+v", r.applyCalls[0], wantRunnerSections)
	}
}

// TestApplyConfig_FailureSetsBannerOnlyAndLeavesEverythingElseUnchanged
// references AC-2 (startup) and AC-3 (reload failure).
func TestApplyConfig_FailureSetsBannerOnlyAndLeavesEverythingElseUnchanged(t *testing.T) {
	u := &fakeUI{}
	r := &fakeRunner{}

	// Establish a baseline successful state first, mirroring a reload that
	// later fails; the failure must change nothing beyond the banner.
	applyConfig(sampleConfig(), nil, u, r)
	windowCallsBefore := len(u.windowSettings)
	sectionCallsBefore := len(u.sections)
	applyCallsBefore := len(r.applyCalls)

	loadErr := errors.New("config: parse boom")
	applyConfig(config.Config{}, loadErr, u, r)

	if len(u.configErrors) != 2 || u.configErrors[1] != loadErr.Error() {
		t.Fatalf("configErrors = %v, want banner set to %q", u.configErrors, loadErr.Error())
	}
	if len(u.windowSettings) != windowCallsBefore {
		t.Errorf("ApplyWindow called on failure: calls = %d, want unchanged %d", len(u.windowSettings), windowCallsBefore)
	}
	if len(u.sections) != sectionCallsBefore {
		t.Errorf("SetSections called on failure: calls = %d, want unchanged %d", len(u.sections), sectionCallsBefore)
	}
	if len(r.applyCalls) != applyCallsBefore {
		t.Errorf("runner.Apply called on failure: calls = %d, want unchanged %d", len(r.applyCalls), applyCallsBefore)
	}
}

// TestApplyConfig_StartupFailureOnlySetsBanner references AC-2 directly: an
// unloadable config at startup (no prior successful load) sets only the
// banner, leaving the UI in its already-empty initial state.
func TestApplyConfig_StartupFailureOnlySetsBanner(t *testing.T) {
	u := &fakeUI{}
	r := &fakeRunner{}

	applyConfig(config.Config{}, errors.New("boom"), u, r)

	if len(u.configErrors) != 1 || u.configErrors[0] != "boom" {
		t.Fatalf("configErrors = %v, want [\"boom\"]", u.configErrors)
	}
	if len(u.windowSettings) != 0 || len(u.sections) != 0 || len(r.applyCalls) != 0 {
		t.Errorf("startup failure touched UI/runner beyond the banner: windowSettings=%v sections=%v applyCalls=%v",
			u.windowSettings, u.sections, r.applyCalls)
	}
}

// --- AC-4: runner event translation and dispatch ---

// TestTranslateEvent references AC-4.
func TestTranslateEvent(t *testing.T) {
	snapshot := func(key string) []string {
		return map[string][]string{
			"stream-key": {"line one", "line two"},
		}[key]
	}

	tests := []struct {
		name       string
		event      runner.Event
		wantKey    string
		wantText   string
		wantErrror bool
	}{
		{
			name:       "replace with valid markup passes through unchanged",
			event:      runner.Event{SectionKey: "clock-key", Kind: runner.KindReplace, Text: "<b>12:00</b>"},
			wantKey:    "clock-key",
			wantText:   "<b>12:00</b>",
			wantErrror: false,
		},
		{
			name:       "replace with invalid markup is escaped",
			event:      runner.Event{SectionKey: "clock-key", Kind: runner.KindReplace, Text: "<b>unclosed"},
			wantKey:    "clock-key",
			wantText:   "&lt;b&gt;unclosed",
			wantErrror: false,
		},
		{
			name:       "append renders the section's buffer snapshot joined as lines",
			event:      runner.Event{SectionKey: "stream-key", Kind: runner.KindAppend, Text: "line two"},
			wantKey:    "stream-key",
			wantText:   "line one\nline two",
			wantErrror: false,
		},
		{
			name:       "error maps to the UI error flag with the message untouched",
			event:      runner.Event{SectionKey: "clock-key", Kind: runner.KindError, Text: "command not found"},
			wantKey:    "clock-key",
			wantText:   "command not found",
			wantErrror: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, text, isError := translateEvent(tt.event, snapshot)
			if key != tt.wantKey {
				t.Errorf("key = %q, want %q (event must target only its own section)", key, tt.wantKey)
			}
			if text != tt.wantText {
				t.Errorf("text = %q, want %q", text, tt.wantText)
			}
			if isError != tt.wantErrror {
				t.Errorf("isError = %v, want %v", isError, tt.wantErrror)
			}
		})
	}
}

// TestConsumeEvents_DispatchesTranslatedUpdates references AC-4: each event
// reaches the UI through the injected dispatch function (a stand-in for
// glib.IdleAdd), translated and targeted at only its own section key.
func TestConsumeEvents_DispatchesTranslatedUpdates(t *testing.T) {
	events := make(chan runner.Event, 3)
	events <- runner.Event{SectionKey: "a", Kind: runner.KindReplace, Text: "hello"}
	events <- runner.Event{SectionKey: "b", Kind: runner.KindError, Text: "boom"}
	close(events)

	snapshot := func(string) []string { return nil }
	u := &fakeUI{}

	var dispatchCount int
	syncDispatch := func(fn func()) {
		dispatchCount++
		fn()
	}

	consumeEvents(events, snapshot, u, syncDispatch)

	if dispatchCount != 2 {
		t.Fatalf("dispatch called %d times, want 2", dispatchCount)
	}
	want := []sectionUpdate{
		{key: "a", markup: "hello", isError: false},
		{key: "b", markup: "boom", isError: true},
	}
	if !reflect.DeepEqual(u.updates, want) {
		t.Errorf("updates = %+v, want %+v", u.updates, want)
	}
}

// --- AC-5: shutdown ordering ---

type orderRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (o *orderRecorder) record(name string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.calls = append(o.calls, name)
}

type orderedWatcher struct{ rec *orderRecorder }

func (w *orderedWatcher) Notify() <-chan struct{} { return nil }
func (w *orderedWatcher) Close() error {
	w.rec.record("watcher.Close")
	return nil
}

type orderedRunner struct{ rec *orderRecorder }

func (r *orderedRunner) Apply(sections []runner.SectionConfig) {}
func (r *orderedRunner) Stop()                                 { r.rec.record("runner.Stop") }
func (r *orderedRunner) Events() <-chan runner.Event           { return nil }
func (r *orderedRunner) Snapshot(key string) []string          { return nil }

// TestApp_Quit_StopsWatcherThenRunnerAndIsIdempotent references AC-5.
func TestApp_Quit_StopsWatcherThenRunnerAndIsIdempotent(t *testing.T) {
	rec := &orderRecorder{}
	a := &App{
		watcher: &orderedWatcher{rec: rec},
		runner:  &orderedRunner{rec: rec},
	}

	a.Quit()
	a.Quit() // idempotent: must not stop twice

	want := []string{"watcher.Close", "runner.Stop"}
	if !reflect.DeepEqual(rec.calls, want) {
		t.Errorf("shutdown order = %v, want %v (exactly once each, watcher before runner)", rec.calls, want)
	}
}

// TestApp_Quit_WithNoWatcherStillStopsRunner covers the case where the
// watcher failed to start (App.Run logs and leaves watcher nil).
func TestApp_Quit_WithNoWatcherStillStopsRunner(t *testing.T) {
	r := &fakeRunner{}
	a := &App{runner: r}

	a.Quit()

	if r.stopCalls != 1 {
		t.Errorf("runner.Stop calls = %d, want 1", r.stopCalls)
	}
}
