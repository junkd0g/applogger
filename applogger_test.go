package applogger_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"sync"
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

// TestLogger_ContextValueExtraction tests that context values are extracted correctly.
func TestLogger_ContextValueExtraction(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	// Create context with applogger_fields map as expected by the implementation
	ctx := context.Background()
	contextFields := map[string]interface{}{
		"user_id":       "12345",
		"request_id":    "req-001",
		"session_id":    "sess-789",
		"numeric_value": 42,
		"bool_value":    true,
	}
	ctx = context.WithValue(ctx, applogger.ApploggerFieldsKey, contextFields)

	logger.Log(ctx, applogger.Info, "Test context extraction")
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatalf("expected at least one log entry")
	}
	entry := entries[len(entries)-1]

	// Check string values
	if entry.Attributes["user_id"] != "12345" {
		t.Errorf("expected user_id=12345, got %v", entry.Attributes["user_id"])
	}
	if entry.Attributes["request_id"] != "req-001" {
		t.Errorf("expected request_id=req-001, got %v", entry.Attributes["request_id"])
	}
	if entry.Attributes["session_id"] != "sess-789" {
		t.Errorf("expected session_id=sess-789, got %v", entry.Attributes["session_id"])
	}

	// Check numeric value (JSON unmarshaling converts to float64)
	if num, ok := entry.Attributes["numeric_value"].(float64); !ok || num != 42 {
		t.Errorf("expected numeric_value=42, got %v", entry.Attributes["numeric_value"])
	}

	// Check boolean value
	if val, ok := entry.Attributes["bool_value"].(bool); !ok || val != true {
		t.Errorf("expected bool_value=true, got %v", entry.Attributes["bool_value"])
	}
}

// TestLogger_ContextValueExtraction_EmptyContext tests logging with empty context.
func TestLogger_ContextValueExtraction_EmptyContext(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	ctx := context.Background()
	logger.Log(ctx, applogger.Info, "Test empty context")
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatalf("expected at least one log entry")
	}
	entry := entries[len(entries)-1]

	// Should have empty attributes map
	if len(entry.Attributes) != 0 {
		t.Errorf("expected empty attributes for empty context, got %v", entry.Attributes)
	}
}

// TestLogger_ContextValueExtraction_NilContext tests logging with nil context.
func TestLogger_ContextValueExtraction_NilContext(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	logger.Log(context.TODO(), applogger.Info, "Test nil context")
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) == 0 {
		t.Fatalf("expected at least one log entry")
	}
	entry := entries[len(entries)-1]

	// Should have empty attributes map
	if len(entry.Attributes) != 0 {
		t.Errorf("expected empty attributes for nil context, got %v", entry.Attributes)
	}
}

// TestLogger_ConcurrentLogging tests thread-safe concurrent logging.
func TestLogger_ConcurrentLogging(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	const numGoroutines = 10
	const logsPerGoroutine = 20
	var wg sync.WaitGroup

	// Launch multiple goroutines to log concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.WithValue(context.Background(), applogger.ApploggerFieldsKey, map[string]interface{}{
				"goroutine_id": id,
			})
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Log(ctx, applogger.Info, "Concurrent log message")
			}
		}(i)
	}

	wg.Wait()
	logger.Close()

	entries := readLogEntries(t, path)
	expectedEntries := numGoroutines * logsPerGoroutine
	if len(entries) != expectedEntries {
		t.Errorf("expected %d log entries, got %d", expectedEntries, len(entries))
	}

	// Verify all entries have the expected message
	for _, entry := range entries {
		if entry.Message != "Concurrent log message" {
			t.Errorf("unexpected message: %s", entry.Message)
		}
		if entry.Level != "INFO" {
			t.Errorf("unexpected level: %s", entry.Level)
		}
	}
}

// TestLogger_ConcurrentHTTPLogging tests concurrent HTTP logging.
func TestLogger_ConcurrentHTTPLogging(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	const numGoroutines = 5
	const logsPerGoroutine = 10
	var wg sync.WaitGroup

	// Launch multiple goroutines for HTTP logging
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.WithValue(context.Background(), applogger.ApploggerFieldsKey, map[string]interface{}{
				"worker_id": id,
			})
			for j := 0; j < logsPerGoroutine; j++ {
				code := 200 + j
				duration := float64(j) * 0.1
				logger.LogHTTP(ctx, applogger.Debug, "HTTP request", code, duration)
			}
		}(i)
	}

	wg.Wait()
	logger.Close()

	entries := readLogEntries(t, path)
	expectedEntries := numGoroutines * logsPerGoroutine
	if len(entries) != expectedEntries {
		t.Errorf("expected %d log entries, got %d", expectedEntries, len(entries))
	}

	// Verify all entries are HTTP logs
	for _, entry := range entries {
		if entry.Message != "HTTP request" {
			t.Errorf("unexpected message: %s", entry.Message)
		}
		if entry.Level != "DEBUG" {
			t.Errorf("unexpected level: %s", entry.Level)
		}
		if entry.Code == 0 {
			t.Errorf("expected non-zero HTTP code")
		}
	}
}

// TestLogger_ConcurrentWithFields tests concurrent logging with WithFields.
func TestLogger_ConcurrentWithFields(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	baseLogger := logger.WithFields(map[string]interface{}{
		"service": "test",
		"version": "1.0",
	})

	const numGoroutines = 5
	var wg sync.WaitGroup

	// Each goroutine creates its own logger with additional fields
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			workerLogger := baseLogger.WithFields(map[string]interface{}{
				"worker_id": id,
			})
			ctx := context.WithValue(context.Background(), applogger.ApploggerFieldsKey, map[string]interface{}{
				"request_id": id * 100,
			})
			workerLogger.Log(ctx, applogger.Info, "Worker message")
		}(i)
	}

	wg.Wait()
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) != numGoroutines {
		t.Errorf("expected %d log entries, got %d", numGoroutines, len(entries))
	}

	// Verify each entry has the expected fields
	for _, entry := range entries {
		if entry.Attributes["service"] != "test" {
			t.Errorf("expected service=test, got %v", entry.Attributes["service"])
		}
		if entry.Attributes["version"] != "1.0" {
			t.Errorf("expected version=1.0, got %v", entry.Attributes["version"])
		}
		if _, exists := entry.Attributes["worker_id"]; !exists {
			t.Errorf("expected worker_id field")
		}
		if _, exists := entry.Attributes["request_id"]; !exists {
			t.Errorf("expected request_id field")
		}
	}
}

// TestLogger_AllLogLevels tests all log levels individually.
func TestLogger_AllLogLevels(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	ctx := context.Background()
	testCases := []struct {
		level    applogger.LogLevel
		expected string
	}{
		{applogger.Debug, "DEBUG"},
		{applogger.Info, "INFO"},
		{applogger.Warn, "WARN"},
		{applogger.Error, "ERROR"},
	}

	for _, tc := range testCases {
		logger.Log(ctx, tc.level, "Test message for "+tc.expected)
	}
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) != len(testCases) {
		t.Fatalf("expected %d log entries, got %d", len(testCases), len(entries))
	}

	for i, entry := range entries {
		if entry.Level != testCases[i].expected {
			t.Errorf("expected level %s, got %s", testCases[i].expected, entry.Level)
		}
		expectedMessage := "Test message for " + testCases[i].expected
		if entry.Message != expectedMessage {
			t.Errorf("expected message %s, got %s", expectedMessage, entry.Message)
		}
	}
}

// TestLogger_LogLevelString tests the String method of LogLevel.
func TestLogger_LogLevelString(t *testing.T) {
	testCases := []struct {
		level    applogger.LogLevel
		expected string
	}{
		{applogger.Debug, "DEBUG"},
		{applogger.Info, "INFO"},
		{applogger.Warn, "WARN"},
		{applogger.Error, "ERROR"},
		{applogger.Fatal, "FATAL"},
		{applogger.LogLevel(999), "UNKNOWN"}, // Invalid log level
	}

	for _, tc := range testCases {
		result := tc.level.String()
		if result != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, result)
		}
	}
}

// TestLogger_HTTPLogLevels tests HTTP logging with different log levels.
func TestLogger_HTTPLogLevels(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	ctx := context.Background()
	testCases := []struct {
		level    applogger.LogLevel
		expected string
		code     int
		duration float64
	}{
		{applogger.Debug, "DEBUG", 200, 0.1},
		{applogger.Info, "INFO", 201, 0.2},
		{applogger.Warn, "WARN", 400, 0.3},
		{applogger.Error, "ERROR", 500, 0.4},
	}

	for _, tc := range testCases {
		logger.LogHTTP(ctx, tc.level, "HTTP test", tc.code, tc.duration)
	}
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) != len(testCases) {
		t.Fatalf("expected %d log entries, got %d", len(testCases), len(entries))
	}

	for i, entry := range entries {
		if entry.Level != testCases[i].expected {
			t.Errorf("expected level %s, got %s", testCases[i].expected, entry.Level)
		}
		if entry.Code != testCases[i].code {
			t.Errorf("expected code %d, got %d", testCases[i].code, entry.Code)
		}
		if entry.Duration != testCases[i].duration {
			t.Errorf("expected duration %f, got %f", testCases[i].duration, entry.Duration)
		}
	}
}

// TestLogger_ErrorHandling tests error scenarios.
func TestLogger_ErrorHandling(t *testing.T) {
	// Test invalid file path
	invalidPath := "/invalid/path/that/does/not/exist/test.log"
	logger, err := applogger.NewLogger(invalidPath)
	if err == nil {
		t.Errorf("expected error for invalid path, got nil")
		if logger != nil {
			logger.Close()
		}
	}
}

// TestLogger_ErrorHandling_ReadOnlyDirectory tests logging to read-only directory.
func TestLogger_ErrorHandling_ReadOnlyDirectory(t *testing.T) {
	// Create a temporary directory and make it read-only
	tempDir, err := os.MkdirTemp("", "readonly_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Make directory read-only
	err = os.Chmod(tempDir, 0444)
	if err != nil {
		t.Fatalf("failed to make directory read-only: %v", err)
	}

	// Restore permissions for cleanup
	defer func() {
		_ = os.Chmod(tempDir, 0755)
	}()

	logPath := tempDir + "/test.log"
	logger, err := applogger.NewLogger(logPath)
	if err == nil {
		t.Errorf("expected error for read-only directory, got nil")
		if logger != nil {
			logger.Close()
		}
	}
}

// TestLogger_FilePermissions tests file permission handling.
func TestLogger_FilePermissions(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	// Log something to create the file
	logger.Log(context.Background(), applogger.Info, "Test message")
	logger.Close()

	// Check that file was created and has content
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat log file: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("expected log file to have content")
	}
}

// TestLogger_EmptyMessage tests logging with empty message.
func TestLogger_EmptyMessage(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	ctx := context.Background()
	logger.Log(ctx, applogger.Info, "")
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Message != "" {
		t.Errorf("expected empty message, got %s", entries[0].Message)
	}
}

// TestLogger_LongMessage tests logging with very long message.
func TestLogger_LongMessage(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	// Create a very long message
	longMessage := strings.Repeat("A", 10000)
	ctx := context.Background()
	logger.Log(ctx, applogger.Info, longMessage)
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	if entries[0].Message != longMessage {
		t.Errorf("message was truncated or modified")
	}
}

// TestLogger_SpecialCharacters tests logging with special characters.
func TestLogger_SpecialCharacters(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	specialMessages := []string{
		"Message with\nnewline",
		"Message with\ttab",
		"Message with \"quotes\"",
		"Message with 'single quotes'",
		"Message with \\backslash",
		"Message with emoji 🚀",
		"Message with unicode: αβγ",
	}

	ctx := context.Background()
	for _, msg := range specialMessages {
		logger.Log(ctx, applogger.Info, msg)
	}
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) != len(specialMessages) {
		t.Fatalf("expected %d log entries, got %d", len(specialMessages), len(entries))
	}

	for i, entry := range entries {
		if entry.Message != specialMessages[i] {
			t.Errorf("expected message %q, got %q", specialMessages[i], entry.Message)
		}
	}
}

// TestLogger_MultipleLoggers tests multiple loggers writing to different files.
func TestLogger_MultipleLoggers(t *testing.T) {
	logger1, path1 := createTempLogger(t)
	logger2, path2 := createTempLogger(t)
	defer func() {
		logger1.Close()
		logger2.Close()
		os.Remove(path1)
		os.Remove(path2)
	}()

	ctx := context.Background()
	logger1.Log(ctx, applogger.Info, "Logger 1 message")
	logger2.Log(ctx, applogger.Info, "Logger 2 message")
	logger1.Close()
	logger2.Close()

	// Check first logger's file
	entries1 := readLogEntries(t, path1)
	if len(entries1) != 1 {
		t.Fatalf("expected 1 log entry in file 1, got %d", len(entries1))
	}
	if entries1[0].Message != "Logger 1 message" {
		t.Errorf("expected 'Logger 1 message', got %s", entries1[0].Message)
	}

	// Check second logger's file
	entries2 := readLogEntries(t, path2)
	if len(entries2) != 1 {
		t.Fatalf("expected 1 log entry in file 2, got %d", len(entries2))
	}
	if entries2[0].Message != "Logger 2 message" {
		t.Errorf("expected 'Logger 2 message', got %s", entries2[0].Message)
	}
}

// TestLogger_WithFieldsOverride tests that WithFields properly overrides existing fields.
func TestLogger_WithFieldsOverride(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	// Create logger with base fields
	baseLogger := logger.WithFields(map[string]interface{}{
		"service": "original",
		"version": "1.0",
	})

	// Override one field and add a new one
	overrideLogger := baseLogger.WithFields(map[string]interface{}{
		"service": "overridden",
		"env":     "test",
	})

	ctx := context.Background()
	overrideLogger.Log(ctx, applogger.Info, "Test override")
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	entry := entries[0]

	// Check that service was overridden
	if entry.Attributes["service"] != "overridden" {
		t.Errorf("expected service=overridden, got %v", entry.Attributes["service"])
	}

	// Check that version was preserved
	if entry.Attributes["version"] != "1.0" {
		t.Errorf("expected version=1.0, got %v", entry.Attributes["version"])
	}

	// Check that new field was added
	if entry.Attributes["env"] != "test" {
		t.Errorf("expected env=test, got %v", entry.Attributes["env"])
	}
}

// TestLogger_PIDFormat tests that PID field format is consistent.
func TestLogger_PIDFormat(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	ctx := context.Background()
	// Log multiple entries with delays to get different PIDs
	for i := 0; i < 3; i++ {
		logger.Log(ctx, applogger.Info, "PID test message")
		time.Sleep(1 * time.Second) // Ensure different second-precision timestamps
	}
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) != 3 {
		t.Fatalf("expected 3 log entries, got %d", len(entries))
	}

	// Check that PID format is consistent (YYYYMMDDHHMMSS)
	for _, entry := range entries {
		if len(entry.PID) != 14 {
			t.Errorf("expected PID length of 14, got %d for PID: %s", len(entry.PID), entry.PID)
		}
		// Try to parse as time to validate format
		_, err := time.Parse("20060102150405", entry.PID)
		if err != nil {
			t.Errorf("PID format is invalid: %s, error: %v", entry.PID, err)
		}
	}
}

// TestLogger_TimestampOrdering tests that timestamps are in chronological order.
func TestLogger_TimestampOrdering(t *testing.T) {
	logger, path := createTempLogger(t)
	defer func() {
		logger.Close()
		os.Remove(path)
	}()

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		logger.Log(ctx, applogger.Info, "Timestamp test")
		time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamps
	}
	logger.Close()

	entries := readLogEntries(t, path)
	if len(entries) != 5 {
		t.Fatalf("expected 5 log entries, got %d", len(entries))
	}

	// Check that timestamps are in chronological order
	for i := 1; i < len(entries); i++ {
		if entries[i].Timestamp.Before(entries[i-1].Timestamp) {
			t.Errorf("timestamps not in chronological order: %v before %v",
				entries[i].Timestamp, entries[i-1].Timestamp)
		}
	}
}

// TestLogger_FatalBehavior tests that Fatal level logging exits the program.
// This test uses a subprocess to avoid exiting the test process.
func TestLogger_FatalBehavior(t *testing.T) {
	// Create a test program that logs with Fatal level
	testProgram := `
package main

import (
	"context"
	"log"
	"os"
	"github.com/junkd0g/applogger"
)

func main() {
	logger, err := applogger.NewLogger("fatal_test.log")
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	
	// This should exit the program
	logger.Log(context.Background(), applogger.Fatal, "Fatal error occurred")
	
	// This should never execute
	logger.Log(context.Background(), applogger.Info, "This should not appear")
}
`

	// Write the test program to a temporary file
	tmpFile, err := os.CreateTemp("", "fatal_test_*.go")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testProgram); err != nil {
		t.Fatalf("failed to write test program: %v", err)
	}
	tmpFile.Close()

	// Run the test program
	cmd := exec.Command("go", "run", tmpFile.Name())
	cmd.Dir = "/Users/iordanispaschalidis/gear/applogger"

	err = cmd.Run()
	if err == nil {
		t.Errorf("expected program to exit with error, but it completed successfully")
	}

	// Check if the exit code indicates program termination
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() == 0 {
			t.Errorf("expected non-zero exit code, got %d", exitError.ExitCode())
		}
	}

	// Check that the log file was created and contains the fatal message
	if _, err := os.Stat("fatal_test.log"); err == nil {
		defer os.Remove("fatal_test.log")

		data, err := os.ReadFile("fatal_test.log")
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		if !strings.Contains(string(data), "Fatal error occurred") {
			t.Errorf("expected fatal message in log file, got: %s", string(data))
		}

		// Check that the info message after fatal is NOT present
		if strings.Contains(string(data), "This should not appear") {
			t.Errorf("found message that should not appear after fatal log")
		}
	}
}

// TestLogger_FatalHTTPBehavior tests that Fatal level HTTP logging exits the program.
func TestLogger_FatalHTTPBehavior(t *testing.T) {
	// Create a test program that logs HTTP with Fatal level
	testProgram := `
package main

import (
	"context"
	"log"
	"os"
	"github.com/junkd0g/applogger"
)

func main() {
	logger, err := applogger.NewLogger("fatal_http_test.log")
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()
	
	// This should exit the program
	logger.LogHTTP(context.Background(), applogger.Fatal, "Fatal HTTP error", 500, 1.0)
	
	// This should never execute
	logger.Log(context.Background(), applogger.Info, "This should not appear")
}
`

	// Write the test program to a temporary file
	tmpFile, err := os.CreateTemp("", "fatal_http_test_*.go")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(testProgram); err != nil {
		t.Fatalf("failed to write test program: %v", err)
	}
	tmpFile.Close()

	// Run the test program
	cmd := exec.Command("go", "run", tmpFile.Name())
	cmd.Dir = "/Users/iordanispaschalidis/gear/applogger"

	err = cmd.Run()
	if err == nil {
		t.Errorf("expected program to exit with error, but it completed successfully")
	}

	// Check if the exit code indicates program termination
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.ExitCode() == 0 {
			t.Errorf("expected non-zero exit code, got %d", exitError.ExitCode())
		}
	}

	// Check that the log file was created and contains the fatal HTTP message
	if _, err := os.Stat("fatal_http_test.log"); err == nil {
		defer os.Remove("fatal_http_test.log")

		data, err := os.ReadFile("fatal_http_test.log")
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		if !strings.Contains(string(data), "Fatal HTTP error") {
			t.Errorf("expected fatal HTTP message in log file, got: %s", string(data))
		}

		// Check that the info message after fatal is NOT present
		if strings.Contains(string(data), "This should not appear") {
			t.Errorf("found message that should not appear after fatal log")
		}
	}
}
