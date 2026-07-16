package runner

import (
	"bytes"
	"context"
	"os/exec"
	"syscall"
	"time"
)

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
// execOnce returns ctx.Err().
func execOnce(ctx context.Context, command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var out bytes.Buffer
	cmd.Stdout = &out

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
	}
}
