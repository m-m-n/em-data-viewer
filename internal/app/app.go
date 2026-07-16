// Package app wires together internal/config, internal/runner and
// internal/ui: CLI startup, the live-reload loop (config watcher → holder →
// runner reconcile → UI), event routing onto the GTK main loop, and clean
// shutdown (see IMPLEMENTATION.md's app layer, FR2/FR3/FR8/FR9/NFR2).
package app

import (
	"log"
	"sync"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"

	"data-viewer/internal/config"
	"data-viewer/internal/runner"
	"data-viewer/internal/ui"
)

// watchDebounce is passed to config.NewWatcher: how long the watcher waits
// for a burst of filesystem events to go quiet before emitting one reload
// notification. Comfortably under the ~1s reload target (NFR2).
const watchDebounce = 300 * time.Millisecond

// UI is the app layer's narrow view of the GTK widget window; *ui.UI
// satisfies it. Narrowing to an interface keeps the wiring/translation
// logic in this package testable with a fake, without pulling GTK into
// tests.
type UI interface {
	ApplyWindow(settings ui.WindowSettings)
	SetSections(sections []ui.SectionViewModel)
	UpdateSection(key, markup string, isError bool)
	SetConfigError(message string)
	Run() int
}

// Runner is the app layer's narrow view of the process runner; *runner.Runner
// satisfies it.
type Runner interface {
	Apply(sections []runner.SectionConfig)
	Stop()
	Events() <-chan runner.Event
	Snapshot(key string) []string
}

// Watcher is the app layer's narrow view of the config file watcher;
// *config.Watcher satisfies it.
type Watcher interface {
	Notify() <-chan struct{}
	Close() error
}

// App wires the config watcher, holder, runner and UI together: the
// construction, startup ordering, event routing and shutdown described in
// the task plan.
type App struct {
	configPath string
	holder     *config.Holder
	runner     Runner
	ui         UI
	watcher    Watcher

	stopOnce sync.Once
}

// ResolveConfigPath resolves the config file path to load (AC-1): an
// explicit CLI argument wins if non-empty; otherwise the XDG-standard user
// config location ($XDG_CONFIG_HOME/data-viewer/config.yaml, falling back
// to ~/.config/data-viewer/config.yaml). Thin wrapper around
// config.DefaultPath so cmd/data-viewer's argument-parsing decision is
// unit-testable here without touching cmd's main().
func ResolveConfigPath(explicit string) (string, error) {
	return config.DefaultPath(explicit)
}

// New constructs an App's collaborators (holder, runner, UI) but performs no
// I/O; call Run to start the watcher/event-consumer goroutines and enter the
// UI's main loop.
func New(configPath string) *App {
	a := &App{
		configPath: configPath,
		holder:     config.NewHolder(),
		runner:     runner.New(),
	}
	a.ui = ui.New(a.Quit)
	return a
}

// Run performs the startup sequence: initial load (failure → empty config +
// banner, AC-2), start the config watcher and the runner-event consumer,
// then block on the UI main loop, returning the process exit code.
func (a *App) Run() int {
	// The GTK frame does not exist until ui.Run's ConnectActivate callback
	// fires, and every UI method silently no-ops while the frame is nil, so
	// the initial reload must be scheduled onto the GTK main loop (running
	// after activation) rather than invoked directly here.
	dispatchOnMainLoop(func() {
		reload(a.configPath, a.holder, a.ui, a.runner)
	})

	w, err := config.NewWatcher(a.configPath, watchDebounce)
	if err != nil {
		// Live reload is unavailable (e.g. the config directory does not
		// exist), but the widget still runs with whatever the initial load
		// produced; this is not fatal.
		log.Printf("app: watch %s: %v", a.configPath, err)
	} else {
		a.watcher = w
		go a.watchLoop()
	}

	go a.eventLoop()

	defer a.Quit()
	return a.ui.Run()
}

// watchLoop reloads the config each time the watcher notifies (FR3), until
// Quit closes the watcher.
func (a *App) watchLoop() {
	for range a.watcher.Notify() {
		dispatchOnMainLoop(func() {
			reload(a.configPath, a.holder, a.ui, a.runner)
		})
	}
}

// eventLoop drains runner events until the Runner closes its channel (Stop
// was called), translating and dispatching each one onto the GTK main loop
// (NFR2: no direct cross-goroutine GTK access).
func (a *App) eventLoop() {
	consumeEvents(a.runner.Events(), a.runner.Snapshot, a.ui, dispatchOnMainLoop)
}

// dispatchOnMainLoop schedules fn to run once on the GTK main loop.
func dispatchOnMainLoop(fn func()) {
	glib.IdleAdd(fn)
}

// Quit stops the watcher and the runner — terminating all child processes
// per the runner's contract — before the UI's own quit handling exits the
// main loop (FR8). Idempotent: a second call is a no-op.
func (a *App) Quit() {
	a.stopOnce.Do(func() {
		if a.watcher != nil {
			a.watcher.Close()
		}
		a.runner.Stop()
	})
}
