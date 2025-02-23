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

// createTempLogger creates a temporary file and returns a logger and the file path.
func createTempLogger(t *testing.T) (*applogger.Logger, string) {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "applogger_test_*.log")
	if err != nil {
		t.Fatal(err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close() // We'll let the logger re-open this file.
	logger, err := applogger.NewLogger(tmpPath)
	if err != nil {
		t.Fatal(err)
	}
	return logger, tmpPath
}

// readLogEntries reads the log file and unmarshals each non-empty line into a LogEntry.
func readLogEntries(t *testing.T, path string) []applogger.LogEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
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
			t.Fatalf("Error unmarshalling log entry: %v\nLine: %s", err, line)
		}
		entries = append(entries, entry)
	}
	return entries
}

// TestLogger_Log tests the basic Log method, ensuring context values are merged.
func TestLogger_Log(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	// Create a context with expected keys.
	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", "12345")
	ctx = context.WithValue(ctx, "request_id", "req-001")
	ctx = context.WithValue(ctx, "session_id", "sess-789")

	message := "Test log message"
	logger.Log(ctx, applogger.Info, message)

	// Allow some time for log to be written and then close.
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatal("Expected at least one log entry, got 0")
	}

	entry := entries[len(entries)-1]
	if entry.Message != message {
		t.Errorf("Expected message %q, got %q", message, entry.Message)
	}
	if entry.Level != "INFO" {
		t.Errorf("Expected level INFO, got %s", entry.Level)
	}
	// Verify that context values were extracted.
	if entry.Attributes["user_id"] != "12345" {
		t.Errorf("Expected user_id=12345, got %v", entry.Attributes["user_id"])
	}
	if entry.Attributes["request_id"] != "req-001" {
		t.Errorf("Expected request_id=req-001, got %v", entry.Attributes["request_id"])
	}
	if entry.Attributes["session_id"] != "sess-789" {
		t.Errorf("Expected session_id=sess-789, got %v", entry.Attributes["session_id"])
	}
}

// TestLogger_LogHTTP tests the LogHTTP method.
func TestLogger_LogHTTP(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	ctx := context.Background()
	message := "HTTP Test"
	code := 200
	duration := 0.123
	logger.LogHTTP(ctx, applogger.Debug, message, code, duration)

	logger.Close()
	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatal("Expected at least one log entry, got 0")
	}
	entry := entries[len(entries)-1]
	if entry.Message != message {
		t.Errorf("Expected message %q, got %q", message, entry.Message)
	}
	if entry.Level != "DEBUG" {
		t.Errorf("Expected level DEBUG, got %s", entry.Level)
	}
	if entry.Code != code {
		t.Errorf("Expected code %d, got %d", code, entry.Code)
	}
	if entry.Duration != duration {
		t.Errorf("Expected duration %f, got %f", duration, entry.Duration)
	}
}

// TestLogger_WithFields tests that WithFields correctly merges default fields.
func TestLogger_WithFields(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	fields := map[string]interface{}{
		"service": "test-service",
		"version": "v1.2.3",
	}
	// Create a new logger with additional default fields.
	newLogger := logger.WithFields(fields)

	ctx := context.Background()
	message := "WithFields test"
	newLogger.Log(ctx, applogger.Warn, message)

	newLogger.Close()
	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatal("Expected at least one log entry, got 0")
	}
	entry := entries[len(entries)-1]
	if entry.Attributes["service"] != "test-service" {
		t.Errorf("Expected service=test-service, got %v", entry.Attributes["service"])
	}
	if entry.Attributes["version"] != "v1.2.3" {
		t.Errorf("Expected version=v1.2.3, got %v", entry.Attributes["version"])
	}
}

// TestLogEntryTimestamp ensures the log entry timestamp falls within an expected range.
func TestLogEntryTimestamp(t *testing.T) {
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
		t.Fatal("Expected at least one log entry, got 0")
	}
	entry := entries[len(entries)-1]
	if entry.Timestamp.Before(start) || entry.Timestamp.After(end) {
		t.Errorf("Timestamp %v not in expected range (%v - %v)", entry.Timestamp, start, end)
	}
}

// TestLogger_MultiWriter checks that the logger writes to both stdout and file.
// Since we cannot capture stdout easily in a cross-platform way, we verify the file output.
func TestLogger_MultiWriter(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	message := "MultiWriter test"
	logger.Log(context.Background(), applogger.Info, message)
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatal("Expected at least one log entry from multiwriter test, got 0")
	}
}
