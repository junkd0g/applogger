package applogger_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/junkd0g/applogger"
)

// createTempLogger creates a temporary logger instance for tests.
func createTempLogger(t *testing.T) (*applogger.Logger, string) {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "applogger_test_*.log")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close() // We'll let NewLogger reopen the file.
	logger, err := applogger.NewLogger(tmpPath)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger, tmpPath
}

// readLogEntries reads the log file and unmarshals each non-empty line into a LogEntry.
func readLogEntries(t *testing.T, path string) []applogger.LogEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	var entries []applogger.LogEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry applogger.LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("failed to unmarshal log entry: %v, line: %s", err, line)
		}
		entries = append(entries, entry)
	}
	return entries
}

// TestLogger_LogHTTP tests the LogHTTP method.
func TestLogger_LogHTTP(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	ctx := context.Background()
	message := "HTTP log test"
	code := 404
	duration := 0.789
	logger.LogHTTP(ctx, applogger.Warn, message, code, duration)
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatalf("expected at least one log entry")
	}
	entry := entries[len(entries)-1]
	if entry.Message != message {
		t.Errorf("expected message %q, got %q", message, entry.Message)
	}
	if entry.Level != "WARN" {
		t.Errorf("expected level WARN, got %s", entry.Level)
	}
	if entry.Code != code {
		t.Errorf("expected code %d, got %d", code, entry.Code)
	}
	if entry.Duration != duration {
		t.Errorf("expected duration %f, got %f", duration, entry.Duration)
	}
}

// TestLogger_WithFields tests that WithFields correctly merges extra default fields.
func TestLogger_WithFields(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	baseFields := map[string]interface{}{
		"base_key": "base_value",
	}
	loggerWithFields := logger.WithFields(baseFields)
	// Merge in additional fields.
	extraFields := map[string]interface{}{
		"extra_key": 123,
	}
	loggerWithFields = loggerWithFields.WithFields(extraFields)

	ctx := context.Background()
	message := "Test WithFields"
	loggerWithFields.Log(ctx, applogger.Debug, message)
	loggerWithFields.Close()

	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatalf("expected at least one log entry")
	}
	entry := entries[len(entries)-1]
	if entry.Level != "DEBUG" {
		t.Errorf("expected level DEBUG, got %s", entry.Level)
	}
	if entry.Attributes["base_key"] != "base_value" {
		t.Errorf("expected base_key=base_value, got %v", entry.Attributes["base_key"])
	}
	// Check the extra field; use float64 for numeric comparison.
	if num, ok := entry.Attributes["extra_key"].(float64); !ok || num != 123 {
		t.Errorf("expected extra_key=123, got %v", entry.Attributes["extra_key"])
	}
}

// TestLogger_Timestamp verifies that the log entry timestamp is within the expected time range.
func TestLogger_Timestamp(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	ctx := context.Background()
	message := "Timestamp test"
	start := time.Now()
	logger.Log(ctx, applogger.Info, message)
	end := time.Now()
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatalf("expected at least one log entry")
	}
	entry := entries[len(entries)-1]
	if entry.Timestamp.Before(start) || entry.Timestamp.After(end) {
		t.Errorf("timestamp %v not within expected range (%v - %v)", entry.Timestamp, start, end)
	}
}

// TestLogger_CloseTwice tests that calling Close twice does not cause a panic.
func TestLogger_CloseTwice(t *testing.T) {
	logger, path := createTempLogger(t)
	defer os.Remove(path)

	if err := logger.Close(); err != nil {
		t.Errorf("first close returned error: %v", err)
	}
	// Second close; should not panic (even if it returns an error).
	_ = logger.Close()
}

// TestGetCallerInfo indirectly tests that getCallerInfo returns valid values.
// This is verified by ensuring that the log entry contains non-"unknown" package/function.
func TestGetCallerInfo(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()
	logger.Log(context.Background(), applogger.Info, "Testing caller info")
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatalf("expected at least one log entry")
	}
	entry := entries[len(entries)-1]
	if entry.Package == "unknown" || entry.Func == "unknown" {
		t.Errorf("expected valid caller info, got package=%q, func=%q", entry.Package, entry.Func)
	}
}
