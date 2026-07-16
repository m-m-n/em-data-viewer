package config

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a config file for changes, debouncing bursts of
// filesystem events into a single notification per change. It
// watches the file's containing DIRECTORY rather than the file
// itself, so editor rename/replace save patterns (write a temp file,
// then rename it over the target) keep working: renaming a new file
// into place is observed as a Create event for the target name in
// the watched directory.
type Watcher struct {
	fsw      *fsnotify.Watcher
	notify   chan struct{}
	done     chan struct{}
	filename string
}

// NewWatcher starts watching the directory containing path for
// changes to path, coalescing bursts of events within debounce into
// a single notification. The directory must exist; the file itself
// need not (a deleted-then-recreated file is still observed). Call
// Close when done to release resources.
func NewWatcher(path string, debounce time.Duration) (*Watcher, error) {
	dir := filepath.Dir(path)

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("config: create watcher: %w", err)
	}
	if err := fsw.Add(dir); err != nil {
		fsw.Close()
		return nil, fmt.Errorf("config: watch %s: %w", dir, err)
	}

	w := &Watcher{
		fsw:      fsw,
		notify:   make(chan struct{}, 1),
		done:     make(chan struct{}),
		filename: filepath.Base(path),
	}
	go w.run(debounce)
	return w, nil
}

// Notify returns the channel on which a value is sent after the
// watched file changes, debounced. The channel is buffered by one;
// a pending notification is not duplicated if the consumer has not
// yet read it.
func (w *Watcher) Notify() <-chan struct{} {
	return w.notify
}

// Close stops the watcher goroutine and releases the underlying
// filesystem watch.
func (w *Watcher) Close() error {
	close(w.done)
	return w.fsw.Close()
}

// run is the watcher's event loop. It filters directory events down
// to Create/Write operations on the target filename, then debounces
// bursts of matching events into a single notification per quiet
// window.
func (w *Watcher) run(debounce time.Duration) {
	var timer *time.Timer
	var timerC <-chan time.Time

	stopTimer := func() {
		if timer != nil {
			timer.Stop()
			timer = nil
			timerC = nil
		}
	}
	defer stopTimer()

	for {
		select {
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}
			if filepath.Base(ev.Name) != w.filename {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(debounce)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(debounce)
			}
			timerC = timer.C

		case <-timerC:
			select {
			case w.notify <- struct{}{}:
			default:
			}
			timerC = nil

		case _, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			// Individual fsnotify errors are not fatal; keep watching.

		case <-w.done:
			return
		}
	}
}
