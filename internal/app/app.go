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

// KeepAboveUI is the app layer's view of the keep-above apply surface
// (ui.UI.SetKeepAbove, task0002's contract per IMPLEMENTATION.md Shared
// Components: "applies EWMH on X11 ... updates the menu check state").
// Declared separately from UI rather than folded into its method set, and
// detected via a type assertion (applyKeepAbove) instead of a direct call:
// *ui.UI does not implement this method yet in this worktree (task0002 has
// not merged), and UI's existing satisfaction by *ui.UI in New() must keep
// compiling independently of task ordering (worktree independence). Once
// ui.UI gains SetKeepAbove for real, applyKeepAbove starts calling it with
// no further change needed here.
type KeepAboveUI interface {
	SetKeepAbove(enabled bool)
}

// applyKeepAbove calls u.SetKeepAbove(enabled) when u implements
// KeepAboveUI, and is a no-op otherwise. Every keep-above apply in this
// package (the reload projection, AC-1/AC-4, and the toggle callback,
// AC-2) is routed through this helper instead of asserting inline.
func applyKeepAbove(u UI, enabled bool) {
	if k, ok := u.(KeepAboveUI); ok {
		k.SetKeepAbove(enabled)
	}
}

// KeepAboveWriter persists the keep-above toggle to the config file;
// config.UpdateKeepAbove (task0001) satisfies it. Declared as a function
// type -- the same seam style as dispatch func(func()) below -- so this
// package's toggle-handling wiring stays unit-testable against a fake
// writer, independently of which config write-back implementation New
// wires in.
type KeepAboveWriter func(path string, enabled bool) error

// handleKeepAboveToggle implements the menu toggle callback (AC-2, AC-3):
// applies the new state immediately -- the callback fires already on the
// GTK main loop (IMPLEMENTATION.md: "invoked from the menu item"), so this
// call needs no dispatch -- then hands persistence to a dedicated
// goroutine, off the main loop (NFR2), so a slow or blocking write cannot
// stall the UI. A write-back failure surfaces through the config-error
// banner, dispatched back onto the main loop, WITHOUT reverting the
// already-applied runtime state (AC-3): the banner is the only side effect
// of a failed write-back.
func handleKeepAboveToggle(enabled bool, configPath string, u UI, write KeepAboveWriter, dispatch func(func())) {
	applyKeepAbove(u, enabled)
	go func() {
		if err := write(configPath, enabled); err != nil {
			dispatch(func() {
				u.SetConfigError(err.Error())
			})
		}
	}()
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
	configPath     string
	holder         *config.Holder
	runner         Runner
	ui             UI
	watcher        Watcher
	store          *sectionStore
	generations    *generationTracker
	writeKeepAbove KeepAboveWriter

	triggers chan struct{}
	done     chan struct{}

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
		configPath:     configPath,
		holder:         config.NewHolder(),
		runner:         runner.New(),
		store:          newSectionStore(),
		generations:    newGenerationTracker(),
		writeKeepAbove: config.UpdateKeepAbove,
		triggers:       make(chan struct{}),
		done:           make(chan struct{}),
	}
	// ui.New does not yet accept the onKeepAboveToggle callback that
	// IMPLEMENTATION.md's Shared Components pins as a constructor parameter
	// (task0002's contract); a.onKeepAboveToggle is ready to be passed once
	// it does (e.g. ui.New(a.Quit, a.onKeepAboveToggle)). Until then the
	// menu toggle has no real trigger to invoke it from, since ui.go/menu.go
	// are not this task's files to change.
	a.ui = ui.New(a.Quit)
	return a
}

// onKeepAboveToggle is the method form of handleKeepAboveToggle, wired to
// this App's collaborators; see New's comment for its pending registration
// with the real UI construction.
func (a *App) onKeepAboveToggle(enabled bool) {
	handleKeepAboveToggle(enabled, a.configPath, a.ui, a.writeKeepAbove, dispatchOnMainLoop)
}

// Run performs the startup sequence: initial load (failure → empty config +
// banner, AC-2), start the config watcher and the runner-event consumer,
// then block on the UI main loop, returning the process exit code.
func (a *App) Run() int {
	// All reload transactions (initial load and every watch notification)
	// are serialized through reloadLoop, a single dedicated goroutine, off
	// the GTK main loop (Rework Round 1/2: reload pipeline, AC-4). The GTK
	// frame does not exist until ui.Run's ConnectActivate callback fires,
	// and every UI method silently no-ops while the frame is nil, so only
	// reload's UI-projection half — dispatched onto the main loop inside
	// reload/applyConfig — needs to wait for activation. Because
	// reloadLoop processes triggers one at a time, in the order they are
	// sent on a.triggers, the generation number, Holder update, Runner.Apply
	// and UI dispatch for a given reload always complete before the next
	// one starts, so trigger order is preserved end-to-end (AC-5).
	go a.reloadLoop()
	a.trigger()

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

// trigger sends a reload trigger to reloadLoop, unless Quit has already
// signaled shutdown via a.done.
func (a *App) trigger() {
	select {
	case a.triggers <- struct{}{}:
	case <-a.done:
	}
}

// reloadLoop consumes reload triggers (the initial load and every watch
// notification) one at a time, in the order they were sent, and runs a full
// reload transaction — generation issuance, load, holder update, runner
// reconcile and UI dispatch — for each before consuming the next. This
// serializes reload transactions in trigger order, so an older reload can
// never overwrite the holder/runner/UI after a newer one (round 2 fix for
// the generation race). It exits once Quit closes a.done.
func (a *App) reloadLoop() {
	for {
		select {
		case <-a.triggers:
			gen := a.generations.next()
			reload(a.configPath, a.holder, a.runner, a.store, gen, a.generations, a.ui, dispatchOnMainLoop)
		case <-a.done:
			return
		}
	}
}

// watchLoop forwards a reload trigger each time the watcher notifies (FR3),
// until Quit closes the watcher and signals a.done.
func (a *App) watchLoop() {
	for range a.watcher.Notify() {
		a.trigger()
	}
}

// eventLoop drains runner events until the Runner closes its channel (Stop
// was called), translating and dispatching each one onto the GTK main loop
// (NFR2: no direct cross-goroutine GTK access).
func (a *App) eventLoop() {
	consumeEvents(a.runner.Events(), a.runner.Snapshot, a.ui, a.store, dispatchOnMainLoop)
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
		if a.done != nil {
			close(a.done)
		}
		if a.watcher != nil {
			a.watcher.Close()
		}
		a.runner.Stop()
	})
}
