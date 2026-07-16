package runner

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"syscall"
)

// runStream spawns command once via the shell, appending each stdout line
// to buf and emitting an append event per line. When the process exits
// for any reason, an error event is emitted and runStream returns; no
// restart is attempted (SPEC FR5). If ctx is cancelled first, the process
// is terminated (graceful signal, then a forced kill after a timeout) and
// runStream returns without emitting an error for that intentional stop.
func runStream(ctx context.Context, key, command string, buf *RingBuffer, events chan<- Event) {
	cmd, stdout, err := startStream(command)
	if err != nil {
		sendEvent(ctx, events, Event{SectionKey: key, Kind: KindError, Text: err.Error()})
		return
	}

	lines := make(chan string)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				err := <-waitErr
				msg := "process exited"
				if err != nil {
					msg = err.Error()
				}
				sendEvent(ctx, events, Event{SectionKey: key, Kind: KindError, Text: msg})
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
