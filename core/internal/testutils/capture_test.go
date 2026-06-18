package testutils

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
)

type fakeFatalHelper struct {
	fatalCalled bool
	fatalArgs   []any
}

func (f *fakeFatalHelper) Helper() {}

func (f *fakeFatalHelper) Fatal(args ...any) {
	f.fatalCalled = true
	f.fatalArgs = args
}

func TestCaptureAndWaitForOutput_ReturnsWhenOutputContainsWant(t *testing.T) {
	originalStdout := os.Stdout

	start := time.Now()
	CaptureAndWaitForOutput(t, "ready", 500*time.Millisecond, func() {
		fmt.Print("service is ready")
	})

	if os.Stdout != originalStdout {
		t.Fatal("expected os.Stdout to be restored after successful capture")
	}

	if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
		t.Fatalf("expected capture to return before timeout, took %s", elapsed)
	}
}

func TestCaptureAndWaitForOutput_CallsFatalOnWrapperError(t *testing.T) {
	originalPipe := osPipe
	t.Cleanup(func() { osPipe = originalPipe })

	osPipe = func() (*os.File, *os.File, error) {
		return nil, nil, errors.New("pipe create failed")
	}

	fake := &fakeFatalHelper{}
	CaptureAndWaitForOutput(fake, "irrelevant", 10*time.Millisecond, func() {})

	if !fake.fatalCalled {
		t.Fatal("expected wrapper to call Fatal when capture returns an error")
	}
}

func TestCaptureAndWaitForOutput_DoesNotCallFatalOnWrapperSuccess(t *testing.T) {
	fake := &fakeFatalHelper{}
	CaptureAndWaitForOutput(fake, "ready", 500*time.Millisecond, func() {
		fmt.Print("ready")
	})

	if fake.fatalCalled {
		t.Fatalf("expected wrapper not to call Fatal on success, got args: %#v", fake.fatalArgs)
	}
}

func TestCaptureAndWaitForOutput_FailsOnTimeoutAndRestoresStdout(t *testing.T) {
	originalStdout := os.Stdout

	err := captureAndWaitForOutput("never-printed", 20*time.Millisecond, func() {
		fmt.Print("something else")
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if os.Stdout != originalStdout {
		t.Fatal("expected os.Stdout to be restored after timeout")
	}
}

func TestCaptureAndWaitForOutput_FailsWhenPipeCreationErrors(t *testing.T) {
	originalPipe := osPipe
	originalStdout := os.Stdout
	t.Cleanup(func() { osPipe = originalPipe })

	osPipe = func() (*os.File, *os.File, error) {
		return nil, nil, errors.New("pipe create failed")
	}

	err := captureAndWaitForOutput("irrelevant", 10*time.Millisecond, func() {})
	if err == nil {
		t.Fatal("expected pipe creation error, got nil")
	}
	if os.Stdout != originalStdout {
		t.Fatal("expected os.Stdout to remain unchanged when pipe creation fails")
	}
}

func TestCaptureAndWaitForOutput_TimeoutMessageIncludesDetails(t *testing.T) {
	err := captureAndWaitForOutput("missing", 15*time.Millisecond, func() {
		fmt.Print("xxxx")
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if got := err.Error(); got != "timed out after 15ms waiting for stdout to contain \"missing\"" {
		t.Fatalf("unexpected timeout error message: %q", got)
	}
}
