package testutils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

var osPipe = os.Pipe

type fatalHelper interface {
	Helper()
	Fatal(args ...any)
}

// CaptureAndWaitForOutput redirects os.Stdout to a pipe, starts fn in a goroutine,
// and blocks until the captured output contains want or the timeout elapses.
// os.Stdout is restored before returning in both cases.
func CaptureAndWaitForOutput(t fatalHelper, want string, timeout time.Duration, fn func()) {
	t.Helper()
	if err := captureAndWaitForOutput(want, timeout, fn); err != nil {
		t.Fatal(err)
	}
}

func captureAndWaitForOutput(want string, timeout time.Duration, fn func()) error {
	old := os.Stdout
	r, w, err := osPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	os.Stdout = w

	go fn()

	found := make(chan struct{})
	go func() {
		buf := make([]byte, 512)
		var accumulated strings.Builder
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				accumulated.Write(buf[:n])
				if strings.Contains(accumulated.String(), want) {
					close(found)
					return
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	select {
	case <-found:
		_ = w.Close()
		os.Stdout = old
	case <-time.After(timeout):
		_ = w.Close()
		os.Stdout = old
		return fmt.Errorf("timed out after %s waiting for stdout to contain %q", timeout, want)
	}

	// Drain any remaining buffered output so the goroutine exits cleanly.
	_, _ = io.Copy(io.Discard, r)
	return nil
}
