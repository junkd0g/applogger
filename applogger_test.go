package applogger

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"testing"
)

// Mock output for testing logs without writing to a file
type mockWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.Write(p)
}

func (m *mockWriter) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.String()
}

func setupTestLogger() (*Logger, *mockWriter) {
	mock := &mockWriter{}
	l := log.New(mock, "", 0)
	return &Logger{logger: l}, mock
}

// Test: Basic Logging Functionality
func TestLogger_Log(t *testing.T) {
	logger, mock := setupTestLogger()

	ctx := context.Background() // Provide a valid context
	logger.Log(ctx, Info, "Test log message")

	var entry LogEntry
	err := json.Unmarshal([]byte(mock.String()), &entry)
	if err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Level != "INFO" || entry.Message != "Test log message" {
		t.Errorf("Unexpected log output: %+v", entry)
	}
}

// Test: HTTP Logging
func TestLogger_LogHTTP(t *testing.T) {
	logger, mock := setupTestLogger()

	ctx := context.Background()
	logger.LogHTTP(ctx, Info, "Test HTTP log", 200, 1.5)

	var entry LogEntry
	err := json.Unmarshal([]byte(mock.String()), &entry)
	if err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Level != "INFO" || entry.Code != 200 || entry.Duration != 1.5 {
		t.Errorf("Unexpected log output: %+v", entry)
	}
}

// Test: Context Key Extraction
func TestLogger_LogWithContext(t *testing.T) {
	logger, mock := setupTestLogger()

	ctx := context.WithValue(context.Background(), "user_id", "12345")
	logger.Log(ctx, Info, "Context log test")

	var entry LogEntry
	err := json.Unmarshal([]byte(mock.String()), &entry)
	if err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	// Ensure the extracted key-value pair is present in attributes
	if value, exists := entry.Attributes["user_id"]; !exists || value != "12345" {
		t.Errorf("Expected user_id=12345 in attributes, got: %+v", entry.Attributes)
	}
}

// Test: Concurrent Logging (Race Condition Check)
func TestLogger_ConcurrentLogging(t *testing.T) {
	logger, mock := setupTestLogger()
	var wg sync.WaitGroup

	const logCount = 100
	wg.Add(logCount)

	for i := 0; i < logCount; i++ {
		go func(i int) {
			defer wg.Done()
			ctx := context.WithValue(context.Background(), "request_id", i)
			logger.Log(ctx, Info, "Concurrent log message")
		}(i)
	}

	wg.Wait()

	// Ensure we have the correct number of log entries
	lines := bytes.Split(mock.buf.Bytes(), []byte("\n"))
	if len(lines)-1 != logCount { // -1 because the last entry may be empty due to newline split
		t.Errorf("Expected %d log entries, got %d", logCount, len(lines)-1)
	}
}

// ✅ Test: Log File Creation
func TestLogger_FileCreation(t *testing.T) {
	tmpFile := "test.log"
	defer os.Remove(tmpFile) // Cleanup after test

	logger, err := NewLogger(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	ctx := context.Background()
	logger.Log(ctx, Info, "File logging test")

	// Check if the file exists
	_, err = os.Stat(tmpFile)
	if os.IsNotExist(err) {
		t.Fatalf("Log file was not created")
	}
}

// Test: Logger Close
func TestLogger_Close(t *testing.T) {
	tmpFile := "test.log"
	defer os.Remove(tmpFile)

	logger, err := NewLogger(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	err = logger.Close()
	if err != nil {
		t.Errorf("Error while closing logger: %v", err)
	}
}

// Test: Fatal Logging (Ensures it exits)
func TestLogger_Fatal(t *testing.T) {
	logger, mock := setupTestLogger()

	// We can't actually exit the test suite, so we check the log output instead
	defer func() {
		if r := recover(); r != nil {
			t.Log("Recovered from os.Exit() to allow testing")
		}
	}()

	ctx := context.Background()
	logger.Log(ctx, Fatal, "Fatal error test")

	var entry LogEntry
	err := json.Unmarshal([]byte(mock.String()), &entry)
	if err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Level != "FATAL" {
		t.Errorf("Expected FATAL log, got: %+v", entry)
	}
}
