package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"syscall"
)

// streamDefaultInitialBufferSize is the scanner's initial per-line
// buffer size; it grows up to streamLineLimit as needed.
const streamDefaultInitialBufferSize = 64 * 1024

// streamLineLimit bounds the maximum length of a single stdout line (a
// scanner token) a stream-mode section will accept before treating it
// as a read error (bufio.ErrTooLong) rather than buffering it without
// bound. A package-level var so tests can shrink it for speed.
var streamLineLimit = 1024 * 1024

// runStream spawns command once via the shell, appending each stdout line
// to buf and emitting an append event per line. When the process exits
// for any reason, an error event is emitted and runStream returns; no
// restart is attempted (SPEC FR5). If a stdout line exceeds
// streamLineLimit (or another scanner read error occurs), the process
// is terminated and reaped rather than left to run indefinitely waiting
// for a natural exit that may never come, and an error event is
// emitted noting the read failure. If ctx is cancelled first, the
// process is terminated (graceful signal, then a forced kill after a
// timeout) and runStream returns without emitting an error for that
// intentional stop.
func runStream(ctx context.Context, key, command string, buf *RingBuffer, events chan<- Event) {
	cmd, stdout, err := startStream(command)
	if err != nil {
		sendEvent(ctx, events, Event{SectionKey: key, Kind: KindError, Text: err.Error()})
		return
	}

	lines := make(chan string)
	scanDone := make(chan error, 1)
	go func() {
		initial := streamDefaultInitialBufferSize
		if streamLineLimit < initial {
			initial = streamLineLimit
		}
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, initial), streamLineLimit)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
		scanDone <- scanner.Err()
	}()

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				if scanErr := <-scanDone; scanErr != nil {
					// The scanner stopped on a read error (e.g. a line
					// exceeding streamLineLimit). The process may still
					// be running (possibly blocked writing to its now-
					// unread stdout pipe), so it must be terminated
					// explicitly rather than waited on.
					terminate(cmd, waitErr)
					sendEvent(ctx, events, Event{SectionKey: key, Kind: KindError, Text: fmt.Sprintf("stream read error: %v", scanErr)})
					return
				}
				select {
				case err := <-waitErr:
					msg := "process exited"
					if err != nil {
						msg = err.Error()
					}
					sendEvent(ctx, events, Event{SectionKey: key, Kind: KindError, Text: msg})
				case <-ctx.Done():
					terminate(cmd, waitErr)
				}
				return
			}
			buf.Append(line)
			sendEvent(ctx, events, Event{SectionKey: key, Kind: KindAppend, Text: line})
		case <-ctx.Done():
			terminate(cmd, waitErr)
			// Drain any remaining scanner sends so that goroutine can
			// exit once the (now-dead) process's stdout reaches EOF.
			go func() {
				for range lines {
				}
			}()
			return
		}
	}
}

func startStream(command string) (*exec.Cmd, io.ReadCloser, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, stdout, nil
}
