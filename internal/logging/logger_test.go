package logging

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestNewWritesJSONLogsToStderrOnly(t *testing.T) {
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	originalStdout := os.Stdout
	originalStderr := os.Stderr
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
	}()

	logger, err := New("info", "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info("hello", slog.String("component", "test"))

	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	if err := stderrWriter.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}

	stdoutOutput, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderrOutput, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	if len(stdoutOutput) != 0 {
		t.Fatalf("stdout output = %q, want empty", string(stdoutOutput))
	}
	if !bytes.Contains(stderrOutput, []byte(`"msg":"hello"`)) {
		t.Fatalf("stderr output = %q, want JSON log message", string(stderrOutput))
	}
	if !bytes.Contains(stderrOutput, []byte(`"component":"test"`)) {
		t.Fatalf("stderr output = %q, want structured field", string(stderrOutput))
	}
}

func TestNewFiltersBelowConfiguredLevel(t *testing.T) {
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	originalStderr := os.Stderr
	os.Stderr = stderrWriter
	defer func() {
		os.Stderr = originalStderr
	}()

	logger, err := New("warn", "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	logger.Info("suppressed")
	logger.Warn("visible")

	if err := stderrWriter.Close(); err != nil {
		t.Fatalf("close stderr writer: %v", err)
	}

	stderrOutput, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	output := string(stderrOutput)
	if strings.Contains(output, "suppressed") {
		t.Fatalf("stderr output contains filtered message: %q", output)
	}
	if !strings.Contains(output, "visible") {
		t.Fatalf("stderr output missing warn message: %q", output)
	}
}

func TestNewRejectsInvalidLevel(t *testing.T) {
	if _, err := New("verbose", ""); err == nil {
		t.Fatal("New() error = nil, want invalid level error")
	}
}
