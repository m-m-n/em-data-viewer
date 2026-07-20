package app

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"em-data-viewer/internal/config"
	"em-data-viewer/internal/runner"
	"em-data-viewer/internal/ui"
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
	keepAboveCalls []bool

	// callOrder records every method call below, in invocation order
	// (window-drag-keep-above AC-1: "ordering within the projection
	// verified"). UpdateSection entries are tagged with their section key
	// since a projection can update several sections.
	callOrder []string
}

func (f *fakeUI) ApplyWindow(settings ui.WindowSettings) {
	f.windowSettings = append(f.windowSettings, settings)
	f.callOrder = append(f.callOrder, "ApplyWindow")
}
func (f *fakeUI) SetSections(sections []ui.SectionViewModel) {
	f.sections = append(f.sections, sections)
	f.callOrder = append(f.callOrder, "SetSections")
}
func (f *fakeUI) UpdateSection(key, markup string, isError bool) {
	f.updates = append(f.updates, sectionUpdate{key, markup, isError})
	f.callOrder = append(f.callOrder, "UpdateSection:"+key)
}
func (f *fakeUI) SetConfigError(message string) {
	f.configErrors = append(f.configErrors, message)
	f.callOrder = append(f.callOrder, "SetConfigError")
}
func (f *fakeUI) Run() int { return 0 }

// SetKeepAbove implements KeepAboveUI (window-drag-keep-above's app-side UI
// seam extension), letting fakeUI stand in for the real *ui.UI in tests of
// applyKeepAbove/handleKeepAboveToggle before task0002 lands.
func (f *fakeUI) SetKeepAbove(enabled bool) {
	f.keepAboveCalls = append(f.keepAboveCalls, enabled)
	f.callOrder = append(f.callOrder, "SetKeepAbove")
}

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

// capturingDispatch stands in for glib.IdleAdd in tests that need to
// control exactly when a dispatched callback runs (AC-2, AC-4, AC-5):
// dispatch only records fn; it never invokes it until the test does.
type capturingDispatch struct {
	calls []func()
}

func (c *capturingDispatch) dispatch(fn func()) {
	c.calls = append(c.calls, fn)
}

// noopGate returns a startupGate with a very long timeout, suitable for the
// many pre-existing applyConfig/reload call sites below that predate the
// startup-window-fit task0001 gate and do not exercise its behavior
// themselves; startup-window-fit-specific gate tests live in gate_test.go
// and construct their own gate directly.
func noopGate() *startupGate {
	return newStartupGate(func() {}, time.Hour)
}

// noopMarkReceived stands in for the gate's markReceived for the many
// pre-existing consumeEvents call sites below that predate the
// startup-window-fit task0001 gate wiring; gate-integration tests
// (AC-2/AC-3/AC-7) live in gate_test.go and pass a real gate's markReceived.
func noopMarkReceived(string) {}

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
		want := filepath.Join("/xdg-home", "em-data-viewer", "config.yaml")
		if got != want {
			t.Errorf("ResolveConfigPath() = %q, want %q", got, want)
		}
	})
}

// --- applyConfig decision logic ---

func sampleConfig() config.Config {
	return config.Config{
		Window: config.Window{Width: 800, Style: map[string]string{"background": "#000"}},
		Sections: []config.SectionConfig{
			{Name: "clock", Command: "date", Mode: config.ModeInterval, Interval: 5 * time.Second, MaxLines: 100},
			{Name: "", Command: "tail -f /tmp/x", Mode: config.ModeStream, MaxLines: 50},
		},
	}
}

// syncDispatch runs fn immediately; a stand-in for glib.IdleAdd in tests
// that do not need to control callback timing.
func syncDispatch(fn func()) { fn() }

func TestApplyConfig_SuccessAppliesWindowSectionsAndRunner(t *testing.T) {
	cfg := sampleConfig()
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, syncDispatch, noopGate())

	if len(u.configErrors) != 1 || u.configErrors[0] != "" {
		t.Fatalf("configErrors = %v, want a single cleared banner", u.configErrors)
	}
	if len(u.windowSettings) != 1 {
		t.Fatalf("windowSettings calls = %d, want 1", len(u.windowSettings))
	}
	wantWindow := ui.WindowSettings{Width: 800, Style: cfg.Window.Style}
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
// references FR9 (last-good behavior).
func TestApplyConfig_FailureSetsBannerOnlyAndLeavesEverythingElseUnchanged(t *testing.T) {
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	// Establish a baseline successful state first, mirroring a reload that
	// later fails; the failure must change nothing beyond the banner.
	applyConfig(sampleConfig(), nil, r, store, gens.next(), gens, u, syncDispatch, noopGate())
	windowCallsBefore := len(u.windowSettings)
	sectionCallsBefore := len(u.sections)
	applyCallsBefore := len(r.applyCalls)

	loadErr := errors.New("config: parse boom")
	applyConfig(config.Config{}, loadErr, r, store, gens.next(), gens, u, syncDispatch, noopGate())

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

// TestApplyConfig_StartupFailureOnlySetsBanner: an unloadable config at
// startup (no prior successful load) sets only the banner, leaving the UI
// in its already-empty initial state.
func TestApplyConfig_StartupFailureOnlySetsBanner(t *testing.T) {
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	applyConfig(config.Config{}, errors.New("boom"), r, store, gens.next(), gens, u, syncDispatch, noopGate())

	if len(u.configErrors) != 1 || u.configErrors[0] != "boom" {
		t.Fatalf("configErrors = %v, want [\"boom\"]", u.configErrors)
	}
	if len(u.windowSettings) != 0 || len(u.sections) != 0 || len(r.applyCalls) != 0 {
		t.Errorf("startup failure touched UI/runner beyond the banner: windowSettings=%v sections=%v applyCalls=%v",
			u.windowSettings, u.sections, r.applyCalls)
	}
}

// --- AC-1: rehydration from the presentation store ---

// TestApplyConfig_RehydratesFromStoreAfterSetSections references AC-1: a
// style-only reload (SetSections rebuilds the section widgets) restores an
// interval section's last output and a failed section's error display
// immediately, from the presentation store — not from the runner, which
// retains neither.
func TestApplyConfig_RehydratesFromStoreAfterSetSections(t *testing.T) {
	cfg := sampleConfig()
	intervalKey := cfg.Sections[0].Key()
	streamKey := cfg.Sections[1].Key()

	store := newSectionStore()
	store.set(intervalKey, "<b>12:00:00</b>", false)
	store.set(streamKey, "boom: command not found", true)

	u := &fakeUI{}
	r := &fakeRunner{} // snapshots left nil: proves rehydration does not depend on Runner.Snapshot
	gens := newGenerationTracker()

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, syncDispatch, noopGate())

	want := []sectionUpdate{
		{key: intervalKey, markup: "<b>12:00:00</b>", isError: false},
		{key: streamKey, markup: "boom: command not found", isError: true},
	}
	if !reflect.DeepEqual(u.updates, want) {
		t.Errorf("rehydration updates = %+v, want %+v", u.updates, want)
	}
}

// TestApplyConfig_NoRehydrationWhenStoreHasNoRecordedState covers a
// section with no prior store entry (never rendered yet): it must be left
// alone, filling in on the next run/event as before.
func TestApplyConfig_NoRehydrationWhenStoreHasNoRecordedState(t *testing.T) {
	cfg := sampleConfig()
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, syncDispatch, noopGate())

	if len(u.updates) != 0 {
		t.Errorf("updates = %+v, want none (store has no recorded state for any section)", u.updates)
	}
}

// --- section-row-break task0001, AC-4: break entries excluded from the
// runner section list and rehydration, but not from the view models ---

// sampleConfigWithBreak returns a config whose Sections contain a break
// entry between two real sections, mirroring sampleConfig's real sections.
func sampleConfigWithBreak() config.Config {
	return config.Config{
		Window: config.Window{Width: 800, Style: map[string]string{"background": "#000"}},
		Sections: []config.SectionConfig{
			{Name: "clock", Command: "date", Mode: config.ModeInterval, Interval: 5 * time.Second, MaxLines: 100},
			{IsBreak: true},
			{Name: "", Command: "tail -f /tmp/x", Mode: config.ModeStream, MaxLines: 50},
		},
	}
}

// TestRunnerSections_ExcludesBreakEntries references AC-4: the runner
// section list contains no break entries.
func TestRunnerSections_ExcludesBreakEntries(t *testing.T) {
	cfg := sampleConfigWithBreak()

	got := runnerSections(cfg.Sections)

	want := []runner.SectionConfig{
		{Key: cfg.Sections[0].Key(), Command: "date", Mode: runner.ModeInterval, Interval: 5 * time.Second, MaxLines: 100},
		{Key: cfg.Sections[2].Key(), Command: "tail -f /tmp/x", Mode: runner.ModeStream, MaxLines: 50},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("runnerSections(...) = %+v, want %+v (break entry excluded)", got, want)
	}
}

// TestRehydrationsFor_ExcludesBreakEntries references AC-4: the
// rehydration list contains no break entries, even when the store happens
// to have an entry under a break's (zero-valued) key.
func TestRehydrationsFor_ExcludesBreakEntries(t *testing.T) {
	cfg := sampleConfigWithBreak()
	store := newSectionStore()
	store.set(cfg.Sections[0].Key(), "<b>12:00:00</b>", false)
	// A break entry's Key() is always the same zero-valued string; if a
	// rogue store entry existed under that key, rehydrationsFor must still
	// never surface it for the break position.
	store.set((config.SectionConfig{}).Key(), "should never surface", false)

	got := rehydrationsFor(cfg.Sections, store)

	want := []sectionRehydration{
		{key: cfg.Sections[0].Key(), text: "<b>12:00:00</b>", isError: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rehydrationsFor(...) = %+v, want %+v (break entry excluded)", got, want)
	}
}

// TestSectionViewModels_MarksBreakEntriesAndPreservesOrder covers the
// conveyance half of AC-4/FR2: unlike the runner list and rehydration, the
// view-model list keeps one entry per config entry (order preserved),
// marking the break entry so the ui layer can branch on it.
func TestSectionViewModels_MarksBreakEntriesAndPreservesOrder(t *testing.T) {
	cfg := sampleConfigWithBreak()

	got := sectionViewModels(cfg.Sections)

	want := []ui.SectionViewModel{
		{Key: cfg.Sections[0].Key(), Label: "clock", Style: cfg.Sections[0].Style},
		{IsBreak: true},
		{Key: cfg.Sections[2].Key(), Label: "", Style: cfg.Sections[2].Style},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sectionViewModels(...) = %+v, want %+v", got, want)
	}
}

// TestApplyConfig_BreakEntryExcludedFromRunnerButKeptInSetSections
// integrates the above through applyConfig: a break-containing config
// still reconciles the runner with only the real sections, while
// SetSections receives one entry per config entry including the break.
func TestApplyConfig_BreakEntryExcludedFromRunnerButKeptInSetSections(t *testing.T) {
	cfg := sampleConfigWithBreak()
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, syncDispatch, noopGate())

	if len(r.applyCalls) != 1 || len(r.applyCalls[0]) != 2 {
		t.Fatalf("runner.Apply got %+v, want exactly 2 sections (break excluded)", r.applyCalls)
	}
	if len(u.sections) != 1 || len(u.sections[0]) != 3 {
		t.Fatalf("SetSections got %+v, want exactly 3 view models (break included)", u.sections)
	}
	if !u.sections[0][1].IsBreak {
		t.Errorf("SetSections view model at the break's position = %+v, want IsBreak = true", u.sections[0][1])
	}
}

// --- AC-4: reconcile off the dispatched path ---

// TestApplyConfig_RunnerReconcileHappensBeforeUIDispatch references AC-4:
// runner.Apply and the view-model/rehydration computation happen
// synchronously — before dispatch's callback is ever invoked — proving
// that work runs off whatever dispatch represents (the GTK main loop in
// production); only the UI method calls themselves wait behind dispatch.
func TestApplyConfig_RunnerReconcileHappensBeforeUIDispatch(t *testing.T) {
	cfg := sampleConfig()
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()
	capture := &capturingDispatch{}

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, capture.dispatch, noopGate())

	if len(r.applyCalls) != 1 {
		t.Fatalf("runner.Apply calls = %d, want 1 (reconcile must not wait for dispatch)", len(r.applyCalls))
	}
	if len(u.windowSettings) != 0 || len(u.sections) != 0 {
		t.Fatalf("UI mutated before dispatch ran: windowSettings=%v sections=%v", u.windowSettings, u.sections)
	}
	if len(capture.calls) != 1 {
		t.Fatalf("dispatch calls = %d, want 1", len(capture.calls))
	}

	capture.calls[0]()

	if len(u.windowSettings) != 1 || len(u.sections) != 1 {
		t.Errorf("UI not updated after dispatch ran: windowSettings=%v sections=%v", u.windowSettings, u.sections)
	}
}

// TestApplyConfig_FailurePathOnlyDispatchesTheBanner references AC-4 for
// the failure branch: nothing runs before dispatch (there is no runner
// reconcile on failure), and only the banner call is behind it.
func TestApplyConfig_FailurePathOnlyDispatchesTheBanner(t *testing.T) {
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()
	capture := &capturingDispatch{}

	applyConfig(config.Config{}, errors.New("boom"), r, store, gens.next(), gens, u, capture.dispatch, noopGate())

	if len(u.configErrors) != 0 {
		t.Fatalf("configErrors = %v before dispatch ran, want none", u.configErrors)
	}
	if len(capture.calls) != 1 {
		t.Fatalf("dispatch calls = %d, want 1", len(capture.calls))
	}

	capture.calls[0]()

	if len(u.configErrors) != 1 || u.configErrors[0] != "boom" {
		t.Errorf("configErrors after dispatch ran = %v, want [\"boom\"]", u.configErrors)
	}
}

// --- startup-window-fit task0004 Fix 1, AC-1/AC-2: projection-before-runner
// ordering ---

// orderingRunner is a Runner whose Apply call records "apply" into a shared
// order slice, letting a test observe when it was invoked relative to other
// recorded actions (task0004 AC-1).
type orderingRunner struct {
	order      *[]string
	applyCalls [][]runner.SectionConfig
}

func (r *orderingRunner) Apply(sections []runner.SectionConfig) {
	*r.order = append(*r.order, "apply")
	r.applyCalls = append(r.applyCalls, sections)
}
func (r *orderingRunner) Stop()                        {}
func (r *orderingRunner) Events() <-chan runner.Event  { return nil }
func (r *orderingRunner) Snapshot(key string) []string { return nil }

// TestApplyConfig_SuccessOrdersProjectionDispatchThenGateArmThenRunnerApply
// references AC-1: in the success path, the projection dispatch, the gate
// arming, and the runner apply are invoked in exactly that order. The
// gate's arm call has no directly observable side effect unless it fires,
// so both of cfg's section keys are marked received before applyConfig
// runs -- gate.go is out of scope for this task, and pre-arm receipts
// reconciling to an empty owed set is the gate's own documented behavior
// (task0003 Fix 1), giving arm() a synchronous, observable "fired" moment
// exactly when applyConfig invokes it.
func TestApplyConfig_SuccessOrdersProjectionDispatchThenGateArmThenRunnerApply(t *testing.T) {
	cfg := sampleConfig()
	u := &fakeUI{}
	store := newSectionStore()
	gens := newGenerationTracker()

	var order []string
	r := &orderingRunner{order: &order}

	g := newStartupGate(func() {
		order = append(order, "arm")
	}, time.Hour)
	g.markReceived(cfg.Sections[0].Key())
	g.markReceived(cfg.Sections[1].Key())

	dispatch := func(fn func()) {
		order = append(order, "dispatch")
	}

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, dispatch, g)

	want := []string{"dispatch", "arm", "apply"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("invocation order = %v, want %v (AC-1: projection dispatch, then gate arm, then runner apply)", order, want)
	}
	if len(r.applyCalls) != 1 {
		t.Fatalf("runner.Apply calls = %d, want 1", len(r.applyCalls))
	}
}

// fifoRunner is a Runner whose Apply call synchronously triggers onApply,
// simulating a runner event whose UI update is scheduled onto the SAME FIFO
// dispatcher applyConfig itself uses, as fast as an event could possibly
// arrive: the instant Apply runs (task0004 AC-2).
type fifoRunner struct {
	onApply func()
}

func (r *fifoRunner) Apply(sections []runner.SectionConfig) {
	if r.onApply != nil {
		r.onApply()
	}
}
func (r *fifoRunner) Stop()                        {}
func (r *fifoRunner) Events() <-chan runner.Event  { return nil }
func (r *fifoRunner) Snapshot(key string) []string { return nil }

// TestApplyConfig_RunnerEventDispatchedTheInstantApplyRunsStillAppliesAfterProjection
// references AC-2: with a FIFO dispatcher, a runner event consumed
// immediately after apply produces its section update after the projection
// callback -- no update is lost to missing widgets. This test fails under
// the pre-task0004 ordering (runner apply before the projection dispatch),
// where the fastest-possible event would be queued ahead of the projection
// that creates the section widgets it targets.
func TestApplyConfig_RunnerEventDispatchedTheInstantApplyRunsStillAppliesAfterProjection(t *testing.T) {
	cfg := sampleConfig()
	key := cfg.Sections[0].Key()
	u := &fakeUI{}
	store := newSectionStore()
	gens := newGenerationTracker()
	capture := &capturingDispatch{}

	r := &fifoRunner{}
	r.onApply = func() {
		capture.dispatch(func() {
			u.UpdateSection(key, "first output", false)
		})
	}

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, capture.dispatch, noopGate())

	if len(capture.calls) != 2 {
		t.Fatalf("dispatch calls = %d, want 2 (the projection, then the fast event's update)", len(capture.calls))
	}

	capture.calls[0]()
	if len(u.sections) != 1 {
		t.Fatalf("running the FIRST dispatched call did not apply the projection (SetSections) -- want it queued before the event update")
	}
	if len(u.updates) != 0 {
		t.Fatalf("event update applied before the projection ran: %+v", u.updates)
	}

	capture.calls[1]()
	if len(u.updates) != 1 || u.updates[0].key != key {
		t.Fatalf("running the SECOND dispatched call did not apply the event update to %q: %+v", key, u.updates)
	}
}

// --- window-drag-keep-above AC-1, AC-4: reload projection routes keep_above ---

// TestApplyConfig_DispatchesSetKeepAboveWithConfigValue references
// window-drag-keep-above AC-1: a successful reload's dispatched projection
// calls SetKeepAbove with the config's current value, ordered alongside the
// other window-level settings (before the section rebuild).
func TestApplyConfig_DispatchesSetKeepAboveWithConfigValue(t *testing.T) {
	cfg := sampleConfig()
	cfg.Window.KeepAbove = true
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, syncDispatch, noopGate())

	if !reflect.DeepEqual(u.keepAboveCalls, []bool{true}) {
		t.Fatalf("keepAboveCalls = %v, want [true]", u.keepAboveCalls)
	}
	wantOrder := []string{"SetConfigError", "ApplyWindow", "SetKeepAbove", "SetSections"}
	if !reflect.DeepEqual(u.callOrder, wantOrder) {
		t.Errorf("callOrder = %v, want %v (SetKeepAbove dispatched alongside window settings, before SetSections)", u.callOrder, wantOrder)
	}
}

// TestApplyConfig_KeepAboveFailurePathDoesNotCallSetKeepAbove references
// FR9/AC-4's interaction with keep_above: a failed load must not call
// SetKeepAbove at all, matching the existing last-good behavior for window
// settings and sections.
func TestApplyConfig_KeepAboveFailurePathDoesNotCallSetKeepAbove(t *testing.T) {
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	applyConfig(config.Config{}, errors.New("boom"), r, store, gens.next(), gens, u, syncDispatch, noopGate())

	if len(u.keepAboveCalls) != 0 {
		t.Errorf("keepAboveCalls = %v, want none (load failure leaves keep_above untouched)", u.keepAboveCalls)
	}
}

// TestApplyConfig_ReloadWithUnchangedKeepAboveValueIsIdempotent references
// window-drag-keep-above AC-4: two successive reloads carrying the same
// keep_above value both call SetKeepAbove with that same value -- app.go
// does not special-case or drop the call when the value is unchanged
// (idempotency at the visual layer is ui.SetKeepAbove's own responsibility,
// task0002; this only proves the app layer routes the current value every
// time, without state churn such as skipping or flipping it).
func TestApplyConfig_ReloadWithUnchangedKeepAboveValueIsIdempotent(t *testing.T) {
	cfg := sampleConfig()
	cfg.Window.KeepAbove = true
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	applyConfig(cfg, nil, r, store, gens.next(), gens, u, syncDispatch, noopGate())
	applyConfig(cfg, nil, r, store, gens.next(), gens, u, syncDispatch, noopGate())

	if !reflect.DeepEqual(u.keepAboveCalls, []bool{true, true}) {
		t.Errorf("keepAboveCalls = %v, want [true true] (same value applied both reloads, no churn)", u.keepAboveCalls)
	}
}

// --- window-drag-keep-above AC-2, AC-3: toggle callback ---

// TestHandleKeepAboveToggle_AppliesImmediatelyAndPersistsOffCaller
// references AC-2: SetKeepAbove is applied before handleKeepAboveToggle
// returns (the callback already runs on the GTK main loop, so no dispatch
// is needed for the apply half), and the write-back runs on its own
// goroutine rather than blocking the caller -- proven by a fake writer that
// blocks until released, well after handleKeepAboveToggle has returned.
func TestHandleKeepAboveToggle_AppliesImmediatelyAndPersistsOffCaller(t *testing.T) {
	u := &fakeUI{}
	release := make(chan struct{})
	calledWith := make(chan struct {
		path    string
		enabled bool
	}, 1)
	write := func(path string, enabled bool) error {
		calledWith <- struct {
			path    string
			enabled bool
		}{path, enabled}
		<-release
		return nil
	}

	handleKeepAboveToggle(true, "/cfg/config.yaml", u, write, syncDispatch)

	if !reflect.DeepEqual(u.keepAboveCalls, []bool{true}) {
		t.Fatalf("keepAboveCalls = %v, want [true] applied before handleKeepAboveToggle returned", u.keepAboveCalls)
	}

	select {
	case got := <-calledWith:
		if got.path != "/cfg/config.yaml" || got.enabled != true {
			t.Errorf("write-back called with (%q, %v), want (\"/cfg/config.yaml\", true)", got.path, got.enabled)
		}
	case <-time.After(time.Second):
		t.Fatal("write-back was never invoked")
	}
	// handleKeepAboveToggle already returned above, before write unblocks --
	// proving it did not wait for the write-back to complete.
	close(release)
}

// TestHandleKeepAboveToggle_FailingWriteBackShowsBannerWithoutReverting
// references AC-3: a failing write-back sets the config-error banner
// (dispatched, per NFR2) and does not revert the already-applied
// SetKeepAbove state.
func TestHandleKeepAboveToggle_FailingWriteBackShowsBannerWithoutReverting(t *testing.T) {
	u := &fakeUI{}
	writeErr := errors.New("config: write boom")
	write := func(path string, enabled bool) error { return writeErr }
	dispatched := make(chan func(), 1)
	dispatch := func(fn func()) { dispatched <- fn }

	handleKeepAboveToggle(true, "/cfg/config.yaml", u, write, dispatch)

	var fn func()
	select {
	case fn = <-dispatched:
	case <-time.After(time.Second):
		t.Fatal("write-back failure was never dispatched")
	}
	fn()

	if len(u.configErrors) != 1 || u.configErrors[0] != writeErr.Error() {
		t.Errorf("configErrors = %v, want [%q]", u.configErrors, writeErr.Error())
	}
	if !reflect.DeepEqual(u.keepAboveCalls, []bool{true}) {
		t.Errorf("keepAboveCalls = %v, want [true] unchanged (a failed write-back must not revert the applied runtime state)", u.keepAboveCalls)
	}
}

// TestHandleKeepAboveToggle_SuccessfulWriteBackDoesNotTouchBanner covers the
// success path of the write-back: no dispatch call at all, so the banner is
// left alone.
func TestHandleKeepAboveToggle_SuccessfulWriteBackDoesNotTouchBanner(t *testing.T) {
	u := &fakeUI{}
	done := make(chan struct{})
	write := func(path string, enabled bool) error {
		close(done)
		return nil
	}
	var dispatchCalls int
	dispatch := func(fn func()) { dispatchCalls++ }

	handleKeepAboveToggle(false, "/cfg/config.yaml", u, write, dispatch)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("write-back was never invoked")
	}

	if dispatchCalls != 0 {
		t.Errorf("dispatch calls = %d, want 0 (no banner on a successful write-back)", dispatchCalls)
	}
	if len(u.configErrors) != 0 {
		t.Errorf("configErrors = %v, want none", u.configErrors)
	}
}

// --- AC-5: stale-generation protection ---

// TestApplyConfig_StaleGenerationDoesNotOverwriteNewer references AC-5: an
// older reload's dispatched callback completing after a newer one has
// already applied must not overwrite the newer state.
func TestApplyConfig_StaleGenerationDoesNotOverwriteNewer(t *testing.T) {
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	oldCfg := config.Config{Window: config.Window{Width: 100}}
	newCfg := config.Config{Window: config.Window{Width: 999}}

	oldGen := gens.next()
	newGen := gens.next()

	var oldCB, newCB func()
	applyConfig(oldCfg, nil, r, store, oldGen, gens, u, func(fn func()) { oldCB = fn }, noopGate())
	applyConfig(newCfg, nil, r, store, newGen, gens, u, func(fn func()) { newCB = fn }, noopGate())

	// The newer reload's callback happens to run first.
	newCB()
	if len(u.windowSettings) != 1 || u.windowSettings[0].Width != 999 {
		t.Fatalf("windowSettings after newer callback = %+v, want a single 999-width entry", u.windowSettings)
	}

	// The older (stale) reload's callback arrives after — must be a no-op.
	oldCB()
	if len(u.windowSettings) != 1 {
		t.Errorf("stale reload overwrote newer state: windowSettings = %+v, want unchanged (still length 1)", u.windowSettings)
	}
}

// TestGenerationTracker_NewerAppliedFirstBlocksOlder is a focused unit
// test of generationTracker itself, references AC-5.
func TestGenerationTracker_NewerAppliedFirstBlocksOlder(t *testing.T) {
	gens := newGenerationTracker()
	older := gens.next()
	newer := gens.next()

	if !gens.markApplied(newer) {
		t.Fatal("markApplied(newer) = false, want true (first apply always succeeds)")
	}
	if gens.markApplied(older) {
		t.Error("markApplied(older) = true after a newer generation applied, want false")
	}
}

func TestGenerationTracker_AppliesInOrderWhenNoRaceOccurs(t *testing.T) {
	gens := newGenerationTracker()
	first := gens.next()
	second := gens.next()

	if !gens.markApplied(first) {
		t.Fatal("markApplied(first) = false, want true")
	}
	if !gens.markApplied(second) {
		t.Error("markApplied(second) = false, want true (strictly newer than the last applied)")
	}
}

// --- AC-6: reload() reads holder as its authoritative source ---

// TestReload_UsesHolderAsAuthoritativeSource references AC-6: reload's UI
// projection is derived from holder.Config()/holder.Err(), read back after
// Update, rather than from a separate write-only local copy of the load
// result.
func TestReload_UsesHolderAsAuthoritativeSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("window:\n  width: 111\n  height: 222\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	holder := config.NewHolder()
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	var captured func()
	reload(path, holder, r, store, gens.next(), gens, u, func(fn func()) { captured = fn }, noopGate())
	captured()

	if err := holder.Err(); err != nil {
		t.Fatalf("holder.Err() = %v, want nil", err)
	}
	if got := holder.Config().Window.Width; got != 111 {
		t.Fatalf("holder.Config().Window.Width = %d, want 111", got)
	}
	if len(u.windowSettings) != 1 || u.windowSettings[0].Width != 111 {
		t.Errorf("windowSettings = %+v, want width 111 (sourced from holder.Config())", u.windowSettings)
	}
}

// TestReload_FailedLoadSetsBannerFromHolderError references AC-6: a failed
// load's banner message comes from holder.Err() after Update.
func TestReload_FailedLoadSetsBannerFromHolderError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")

	holder := config.NewHolder()
	u := &fakeUI{}
	r := &fakeRunner{}
	store := newSectionStore()
	gens := newGenerationTracker()

	var captured func()
	reload(path, holder, r, store, gens.next(), gens, u, func(fn func()) { captured = fn }, noopGate())
	captured()

	if holder.Err() == nil {
		t.Fatal("holder.Err() = nil, want the load failure recorded")
	}
	if len(u.configErrors) != 1 || u.configErrors[0] != holder.Err().Error() {
		t.Errorf("configErrors = %v, want [%q] (sourced from holder.Err())", u.configErrors, holder.Err().Error())
	}
}

// --- runner event translation and dispatch ---

// TestTranslateEvent covers translateEvent's per-Kind mapping.
func TestTranslateEvent(t *testing.T) {
	snapshot := func(key string) []string {
		return map[string][]string{
			"stream-key":              {"line one", "line two"},
			"stream-trailing-blanks":  {"line one", "line two", ""},
			"stream-only-blank-lines": {"", "", ""},
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
		// --- AC-1: single trailing newline stripped (replace) ---
		{
			name:       "AC-1: replace with a single trailing newline is stripped",
			event:      runner.Event{SectionKey: "clock-key", Kind: runner.KindReplace, Text: "12:00\n"},
			wantKey:    "clock-key",
			wantText:   "12:00",
			wantErrror: false,
		},
		// --- AC-2: multiple trailing newlines all stripped (replace) ---
		{
			name:       "AC-2: replace with multiple trailing newlines has all of them stripped",
			event:      runner.Event{SectionKey: "clock-key", Kind: runner.KindReplace, Text: "12:00\n\n\n"},
			wantKey:    "clock-key",
			wantText:   "12:00",
			wantErrror: false,
		},
		// --- AC-3: append snapshot with trailing empty lines strips the join ---
		{
			name:       "AC-3: append snapshot ending in empty lines renders with no trailing newlines",
			event:      runner.Event{SectionKey: "stream-trailing-blanks", Kind: runner.KindAppend, Text: ""},
			wantKey:    "stream-trailing-blanks",
			wantText:   "line one\nline two",
			wantErrror: false,
		},
		{
			name:       "AC-3/AC-6: append snapshot of only empty lines renders as the empty string",
			event:      runner.Event{SectionKey: "stream-only-blank-lines", Kind: runner.KindAppend, Text: ""},
			wantKey:    "stream-only-blank-lines",
			wantText:   "",
			wantErrror: false,
		},
		// --- AC-4: error message's trailing newline stripped before the UI's "error: " prefix ---
		{
			name:       "AC-4: error message with a trailing newline is stripped",
			event:      runner.Event{SectionKey: "clock-key", Kind: runner.KindError, Text: "command not found\n"},
			wantKey:    "clock-key",
			wantText:   "command not found",
			wantErrror: true,
		},
		// --- AC-5: interior blank line preserved, only trailing newlines removed ---
		{
			name:       "AC-5: interior blank line is kept, only the trailing newline is stripped",
			event:      runner.Event{SectionKey: "clock-key", Kind: runner.KindReplace, Text: "line1\n\nline2\n"},
			wantKey:    "clock-key",
			wantText:   "line1\n\nline2",
			wantErrror: false,
		},
		// --- AC-6: text of only newlines becomes the empty string ---
		{
			name:       "AC-6: replace text consisting only of newlines becomes the empty string",
			event:      runner.Event{SectionKey: "clock-key", Kind: runner.KindReplace, Text: "\n\n\n"},
			wantKey:    "clock-key",
			wantText:   "",
			wantErrror: false,
		},
		// --- AC-7: trailing CRLF fully removed; interior CR untouched ---
		{
			name:       "AC-7: trailing carriage-return + newline pair is fully removed, interior CR untouched",
			event:      runner.Event{SectionKey: "clock-key", Kind: runner.KindReplace, Text: "abc\rdef\r\n"},
			wantKey:    "clock-key",
			wantText:   "abc\rdef",
			wantErrror: false,
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

// TestTranslateEvent_AppendDoesNotMutateSnapshotSource references AC-3's
// second half: stripping trailing newlines from the joined display text
// must not mutate the buffer's stored lines returned by snapshot.
func TestTranslateEvent_AppendDoesNotMutateSnapshotSource(t *testing.T) {
	source := []string{"line one", "line two", ""}
	original := append([]string(nil), source...)
	snapshot := func(key string) []string { return source }

	_, text, _ := translateEvent(runner.Event{SectionKey: "s", Kind: runner.KindAppend, Text: "line two"}, snapshot)

	if text != "line one\nline two" {
		t.Errorf("text = %q, want %q", text, "line one\nline two")
	}
	if !reflect.DeepEqual(source, original) {
		t.Errorf("snapshot source mutated: got %v, want unchanged %v", source, original)
	}
}

// TestConsumeEvents_DispatchesTranslatedUpdates: each event reaches the UI
// through the injected dispatch function (a stand-in for glib.IdleAdd),
// translated and targeted at only its own section key.
func TestConsumeEvents_DispatchesTranslatedUpdates(t *testing.T) {
	events := make(chan runner.Event, 3)
	events <- runner.Event{SectionKey: "a", Kind: runner.KindReplace, Text: "hello"}
	events <- runner.Event{SectionKey: "b", Kind: runner.KindError, Text: "boom"}
	close(events)

	snapshot := func(string) []string { return nil }
	u := &fakeUI{}
	store := newSectionStore()

	var dispatchCount int
	countingDispatch := func(fn func()) {
		dispatchCount++
		fn()
	}

	consumeEvents(events, snapshot, u, store, countingDispatch, noopMarkReceived)

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

// TestConsumeEvents_WritesToStore references AC-1's write side: every
// dispatched update must land in the presentation store under its
// section's key, for the reload path to read back later.
func TestConsumeEvents_WritesToStore(t *testing.T) {
	events := make(chan runner.Event, 1)
	events <- runner.Event{SectionKey: "clock-key", Kind: runner.KindReplace, Text: "12:00"}
	close(events)

	u := &fakeUI{}
	store := newSectionStore()

	consumeEvents(events, func(string) []string { return nil }, u, store, syncDispatch, noopMarkReceived)

	st, ok := store.get("clock-key")
	if !ok || st.text != "12:00" || st.isError {
		t.Errorf("store.get(%q) = %+v, %v, want {text: \"12:00\", isError: false}, true", "clock-key", st, ok)
	}
}

// --- AC-2: error preservation across coalescing ---

// TestConsumeEvents_ErrorIsRenderedBeforeALaterSuccessTakesItsPlace
// references AC-2: a success event for the same section arriving before
// the error's dispatch callback has run must not silently discard the
// error — the error is rendered first, and the success is rendered after.
func TestConsumeEvents_ErrorIsRenderedBeforeALaterSuccessTakesItsPlace(t *testing.T) {
	events := make(chan runner.Event, 2)
	events <- runner.Event{SectionKey: "a", Kind: runner.KindError, Text: "boom"}
	events <- runner.Event{SectionKey: "a", Kind: runner.KindReplace, Text: "ok again"}
	close(events)

	u := &fakeUI{}
	store := newSectionStore()
	capture := &capturingDispatch{}

	consumeEvents(events, func(string) []string { return nil }, u, store, capture.dispatch, noopMarkReceived)

	if len(capture.calls) != 1 {
		t.Fatalf("dispatch calls after both events received = %d, want 1 (error still pending; success queued behind it, not a second callback yet)", len(capture.calls))
	}
	if len(u.updates) != 0 {
		t.Fatalf("UpdateSection called before any dispatch callback ran: %+v", u.updates)
	}

	capture.calls[0]() // renders the error, then reschedules the queued success

	if len(capture.calls) != 2 {
		t.Fatalf("dispatch calls after running the first callback = %d, want 2 (queued success rescheduled)", len(capture.calls))
	}
	if len(u.updates) != 1 || !u.updates[0].isError {
		t.Fatalf("updates after first callback = %+v, want exactly the error rendered", u.updates)
	}

	capture.calls[1]()

	want := []sectionUpdate{
		{key: "a", markup: "boom", isError: true},
		{key: "a", markup: "ok again", isError: false},
	}
	if !reflect.DeepEqual(u.updates, want) {
		t.Errorf("updates = %+v, want %+v (error rendered at least once, before the success)", u.updates, want)
	}
}

// TestConsumeEvents_SuccessEventsCoalesceIntoOneCallback covers the
// ordinary (non-error) coalescing path still bounded to one pending
// dispatch callback per section.
func TestConsumeEvents_SuccessEventsCoalesceIntoOneCallback(t *testing.T) {
	events := make(chan runner.Event, 3)
	events <- runner.Event{SectionKey: "a", Kind: runner.KindReplace, Text: "one"}
	events <- runner.Event{SectionKey: "a", Kind: runner.KindReplace, Text: "two"}
	events <- runner.Event{SectionKey: "a", Kind: runner.KindReplace, Text: "three"}
	close(events)

	u := &fakeUI{}
	store := newSectionStore()
	capture := &capturingDispatch{}

	consumeEvents(events, func(string) []string { return nil }, u, store, capture.dispatch, noopMarkReceived)

	if len(capture.calls) != 1 {
		t.Fatalf("dispatch calls = %d, want 1 (three same-section events coalesce)", len(capture.calls))
	}

	capture.calls[0]()

	want := []sectionUpdate{{key: "a", markup: "three", isError: false}}
	if !reflect.DeepEqual(u.updates, want) {
		t.Errorf("updates = %+v, want %+v (only the latest coalesced event is rendered)", u.updates, want)
	}
}

// --- AC-3: render/snapshot at dispatch time, not receive time ---

// TestConsumeEvents_RendersOnlyAtDispatchTime references AC-3: rapid
// events for one section coalesce into a single pending dispatch callback,
// and the snapshot (part of the render pipeline) only runs when that
// callback actually fires — never at receive time — so dropped
// intermediate events incur no render.
func TestConsumeEvents_RendersOnlyAtDispatchTime(t *testing.T) {
	events := make(chan runner.Event, 3)
	events <- runner.Event{SectionKey: "s", Kind: runner.KindAppend, Text: "line one"}
	events <- runner.Event{SectionKey: "s", Kind: runner.KindAppend, Text: "line two"}
	events <- runner.Event{SectionKey: "s", Kind: runner.KindAppend, Text: "line three"}
	close(events)

	u := &fakeUI{}
	store := newSectionStore()
	capture := &capturingDispatch{}
	var snapshotCalls int
	snapshot := func(string) []string {
		snapshotCalls++
		return []string{"line one", "line two", "line three"}
	}

	consumeEvents(events, snapshot, u, store, capture.dispatch, noopMarkReceived)

	if snapshotCalls != 0 {
		t.Fatalf("snapshot called %d times before any dispatch callback ran, want 0 (render must not happen at receive time)", snapshotCalls)
	}
	if len(capture.calls) != 1 {
		t.Fatalf("dispatch calls = %d, want 1 (three events coalesce into one pending callback)", len(capture.calls))
	}

	capture.calls[0]()

	if snapshotCalls != 1 {
		t.Errorf("snapshot called %d times after the single dispatch callback ran, want exactly 1", snapshotCalls)
	}
	want := []sectionUpdate{{key: "s", markup: "line one\nline two\nline three", isError: false}}
	if !reflect.DeepEqual(u.updates, want) {
		t.Errorf("updates = %+v, want %+v", u.updates, want)
	}
}

// --- shutdown ordering ---

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

// TestApp_Quit_StopsWatcherThenRunnerAndIsIdempotent covers App.Quit's
// shutdown ordering (FR8).
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
