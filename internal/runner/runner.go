package runner

import (
	"context"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Mode identifies how a section's command is executed.
type Mode string

const (
	ModeInterval Mode = "interval"
	ModeStream   Mode = "stream"
)

// SectionConfig is the minimal section definition the runner acts on. It
// is derived by the caller (the app wiring layer) from the user-facing
// config; the runner package itself has no dependency on that schema.
type SectionConfig struct {
	// Key identifies the section. Sections with the same Key across two
	// Apply calls are left running untouched; a different Key is treated
	// as removed+added (a "restart" from the user's point of view, since
	// Key changes whenever command/mode/interval/max_lines change).
	Key      string
	Command  string
	Mode     Mode
	Interval time.Duration
	MaxLines int
}

// eventBuffer bounds the Runner's event channel so a stalled consumer
// cannot make the Runner grow memory unboundedly; sends simply block
// (backpressure) once it is full.
const eventBuffer = 64

// terminateGrace is how long a section's process is given to exit after a
// graceful termination signal before it is forcefully killed. A
// package-level var so tests can shorten it.
var terminateGrace = 200 * time.Millisecond

type section struct {
	cancel context.CancelFunc
	done   chan struct{}
	buf    *RingBuffer // non-nil for stream-mode sections
}

// Runner executes configured sections (interval or stream mode),
// delivering output/error events on a single channel, and reconciles the
// running set against config changes via Apply.
type Runner struct {
	mu       sync.Mutex
	sections map[string]*section
	events   chan Event
	stopped  bool
}

// New creates a Runner with no sections running. Call Apply to start
// sections and Stop to shut everything down.
func New() *Runner {
	return &Runner{
		sections: make(map[string]*section),
		events:   make(chan Event, eventBuffer),
	}
}

// Events returns the channel on which output/error events are delivered.
// It is closed once Stop has terminated all sections.
func (r *Runner) Events() <-chan Event {
	return r.events
}

// Apply reconciles the running section set against sections: keys not
// present before are started, keys no longer present are stopped, and
// keys present in both are left running untouched. A section whose
// command/mode/interval/max_lines changed gets a different Key (per the
// shared Key contract), so this same diff naturally stops the old key and
// starts the new one. Apply is a no-op after Stop.
func (r *Runner) Apply(sections []SectionConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return
	}

	wanted := make(map[string]SectionConfig, len(sections))
	for _, s := range sections {
		wanted[s.Key] = s
	}

	for key, sec := range r.sections {
		if _, ok := wanted[key]; !ok {
			stopSection(sec)
			delete(r.sections, key)
		}
	}

	for _, cfg := range sections {
		if _, ok := r.sections[cfg.Key]; ok {
			continue
		}
		r.sections[cfg.Key] = r.startSection(cfg)
	}
}

// Stop terminates all running child processes (graceful signal, then a
// forced kill after a timeout) and closes the event channel. Safe to call
// once; a later call is a no-op.
func (r *Runner) Stop() {
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	r.stopped = true
	sections := make([]*section, 0, len(r.sections))
	for _, sec := range r.sections {
		sections = append(sections, sec)
	}
	r.sections = make(map[string]*section)
	r.mu.Unlock()

	for _, sec := range sections {
		sec.cancel()
	}
	for _, sec := range sections {
		<-sec.done
	}
	close(r.events)
}

// Snapshot returns the current buffered lines, in order, for a
// stream-mode section. It returns nil for an unknown key or a non-stream
// section.
func (r *Runner) Snapshot(key string) []string {
	r.mu.Lock()
	sec, ok := r.sections[key]
	r.mu.Unlock()
	if !ok || sec.buf == nil {
		return nil
	}
	return sec.buf.Snapshot()
}

func (r *Runner) startSection(cfg SectionConfig) *section {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	sec := &section{cancel: cancel, done: done}

	if cfg.Mode == ModeStream {
		maxLines := cfg.MaxLines
		if maxLines < 1 {
			maxLines = 1
		}
		sec.buf = NewRingBuffer(maxLines)
		go func() {
			defer close(done)
			runStream(ctx, cfg.Key, cfg.Command, sec.buf, r.events)
		}()
		return sec
	}

	go func() {
		defer close(done)
		runInterval(ctx, cfg.Key, cfg.Command, cfg.Interval, r.events)
	}()
	return sec
}

func stopSection(sec *section) {
	sec.cancel()
	<-sec.done
}

// sendEvent delivers ev on events, or drops it if ctx is done first (the
// section is shutting down and nobody will observe it).
func sendEvent(ctx context.Context, events chan<- Event, ev Event) {
	select {
	case events <- ev:
	case <-ctx.Done():
	}
}

// terminate sends a graceful termination signal to cmd's whole process
// group, waiting up to terminateGrace for it to exit before forcefully
// killing the group. waitErr must receive cmd.Wait()'s result exactly
// once, from a goroutine already running.
func terminate(cmd *exec.Cmd, waitErr <-chan error) {
	pgid := cmd.Process.Pid
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	select {
	case <-waitErr:
		return
	case <-time.After(terminateGrace):
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	<-waitErr
}
