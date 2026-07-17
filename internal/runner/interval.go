package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// intervalOutputLimit bounds how much stdout a single interval-mode run
// may capture before it is terminated as an error, so a runaway command
// (e.g. one that never stops producing output) cannot grow the
// process's memory without bound. A package-level var so tests can
// shrink it for speed.
var intervalOutputLimit = 1 << 20 // 1 MiB

// errOutputLimitExceeded is returned by limitWriter.Write once the
// configured limit has been exceeded.
var errOutputLimitExceeded = errors.New("output exceeded limit")

// limitWriter accumulates written bytes up to limit. Once a write would
// exceed the limit, Write starts returning errOutputLimitExceeded
// instead of writing (which stops Go's internal copy from the child's
// stdout pipe), and overflow is closed exactly once so a caller
// select()ing on it can react immediately: the child itself is not
// guaranteed to exit on its own (it may block writing to its now-full
// stdout pipe, or keep producing output indefinitely), so the caller
// must terminate it explicitly rather than waiting for a natural exit.
type limitWriter struct {
	mu             sync.Mutex
	buf            bytes.Buffer
	limit          int
	overflowed     bool
	overflow       chan struct{}
	overflowClosed sync.Once
}

func newLimitWriter(limit int) *limitWriter {
	return &limitWriter{limit: limit, overflow: make(chan struct{})}
}

func (w *limitWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.overflowed || w.buf.Len()+len(p) > w.limit {
		w.overflowed = true
		w.overflowClosed.Do(func() { close(w.overflow) })
		return 0, errOutputLimitExceeded
	}
	return w.buf.Write(p)
}

func (w *limitWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// runInterval runs command via the shell on a fixed schedule until ctx is
// cancelled: each run emits a replace event with the command's full
// stdout on success, or an error event on failure (spawn failure or
// non-zero exit); the cycle continues after a failure.
func runInterval(ctx context.Context, key, command string, interval time.Duration, events chan<- Event) {
	runOnce := func() {
		out, err := execOnce(ctx, command)
		if err != nil {
			if ctx.Err() != nil {
				// The run was cut short by our own cancellation
				// (Stop/Apply), not a real command failure.
				return
			}
			sendEvent(ctx, events, Event{SectionKey: key, Kind: KindError, Text: err.Error()})
			return
		}
		sendEvent(ctx, events, Event{SectionKey: key, Kind: KindReplace, Text: out})
	}

	runOnce()

	if interval <= 0 {
		<-ctx.Done()
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runOnce()
		}
	}
}

// execOnce runs command via `sh -c` once, returning its captured stdout.
// If ctx is cancelled while the command is running, the command is
// terminated (graceful signal, then a forced kill after a timeout) and
// execOnce returns ctx.Err(). If the command's stdout exceeds
// intervalOutputLimit, the command is terminated the same way and
// execOnce returns a descriptive error noting the truncation.
func execOnce(ctx context.Context, command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	out := newLimitWriter(intervalOutputLimit)
	cmd.Stdout = out

	if err := cmd.Start(); err != nil {
		return "", err
	}

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()

	select {
	case err := <-waitErr:
		if err != nil {
			return "", err
		}
		return out.String(), nil
	case <-ctx.Done():
		terminate(cmd, waitErr)
		return "", ctx.Err()
	case <-out.overflow:
		terminate(cmd, waitErr)
		return "", fmt.Errorf("output exceeded limit of %d bytes; run terminated", out.limit)
	}
}
