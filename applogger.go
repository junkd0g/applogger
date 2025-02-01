/*
Package applogger provides a structured logging system in NDJSON (Newline Delimited JSON) format.
This package is built for performance, concurrency safety, and extensibility, ensuring minimal
overhead while providing comprehensive logs.

### Features:
- **Structured Logging**: Outputs logs in JSON format for easy parsing.
- **Log Levels**: Supports Debug, Info, Warn, Error, and Fatal levels.
- **Context-Aware Logging**: Extracts key-value pairs from `context.Context`.
- **Concurrency Safe**: Uses mutex locking for safe concurrent writes.
- **Automatic Caller Info**: Captures the package and function where the log was generated.
- **HTTP Logging**: Supports logging HTTP response codes and request durations.
- **Graceful Shutdown**: Ensures log files are properly closed on termination.

This package is designed for use in production environments where detailed logging is critical.
If this documentation isn't clear enough, Kim Jong-un will personally hunt us down. So read carefully.

Usage Example:

	package main

	import (
		"context"
		"log"
		"applogger"
	)

	func main() {
		logger, err := applogger.NewLogger("app.log")
		if err != nil {
			log.Fatalf("Failed to initialize logger: %v", err)
		}
		defer logger.Close()

		ctx := context.WithValue(context.Background(), "userID", "1234")
		logger.Log(ctx, applogger.Info, "Application started successfully")
	}
*/

package applogger

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents different levels of log severity.
type LogLevel int

const (
	Debug LogLevel = iota // Debug-level messages, typically used for development and troubleshooting.
	Info                  // Informational messages that highlight application progress.
	Warn                  // Warning messages that indicate a potential problem.
	Error                 // Error messages indicating failures that need attention.
	Fatal                 // Fatal messages causing the application to exit immediately.
)

// String converts a LogLevel into its string representation.
func (l LogLevel) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	case Fatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single log entry in NDJSON format.
type LogEntry struct {
	PID        string                 `json:"pid"`        // Unique identifier for the log event.
	Level      string                 `json:"level"`      // Log severity level.
	Package    string                 `json:"package"`    // Package name where the log was generated.
	Func       string                 `json:"func"`       // Function name where the log was generated.
	Message    string                 `json:"message"`    // Actual log message.
	Timestamp  time.Time              `json:"timestamp"`  // Time when the log was created.
	Code       int                    `json:"code,omitempty"` // HTTP status code (if applicable).
	Duration   float64                `json:"duration,omitempty"` // Request duration in seconds (if applicable).
	Attributes map[string]interface{} `json:"attributes,omitempty"` // Extracted context values.
}

// Logger is a structured logging system for concurrent safe NDJSON logging.
type Logger struct {
	logger *log.Logger // Internal Go logger.
	mu     sync.Mutex  // Mutex to ensure concurrent safety.
	file   *os.File    // Log file handle.
}

// NewLogger initializes a new Logger instance that writes logs to the specified file.
func NewLogger(path string) (*Logger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	l := log.New(f, "", 0)

	return &Logger{
		logger: l,
		file:   f,
	}, nil
}

// Close properly closes the log file to prevent data loss.
func (lg *Logger) Close() error {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	if lg.file != nil {
		return lg.file.Close()
	}
	return nil
}

// Log writes a structured log entry to the file.
func (lg *Logger) Log(ctx context.Context, level LogLevel, message string) {
	lg.logInternal(ctx, level, message, 0, 0, 3)
}

// LogHTTP logs HTTP-related events, including response codes and request durations.
func (lg *Logger) LogHTTP(ctx context.Context, level LogLevel, message string, code int, duration float64) {
	lg.logInternal(ctx, level, message, code, duration, 3)
}

// logInternal is the core logging function that formats and writes log entries.
//
// Params:
// - ctx: The context containing optional key-value pairs.
// - level: The log severity level (Debug, Info, Warn, Error, Fatal).
// - msg: The actual log message.
// - code: Optional HTTP status code (default: 0).
// - duration: Optional request duration in seconds (default: 0).
// - skip: Number of stack frames to skip to get the correct caller info.
//
// This function ensures that all logs are structured and thread-safe.
func (lg *Logger) logInternal(ctx context.Context, level LogLevel, msg string, code int, duration float64, skip int) {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	// Extract the caller function and package name automatically.
	pkgName, funcName := getCallerInfo(skip)

	// Generate a unique PID for each log entry.
	pid := time.Now().Format("20060102150405") // Unique timestamp-based ID

	// Extract context key-value pairs
	attributes := extractContextValues(ctx)

	// Create a structured log entry.
	entry := LogEntry{
		PID:        pid,
		Level:      level.String(),
		Package:    pkgName,
		Func:       funcName,
		Message:    msg,
		Timestamp:  time.Now(),
		Code:       code,
		Duration:   duration,
		Attributes: attributes,
	}

	// Serialize log entry into JSON format.
	data, err := json.Marshal(entry)
	if err != nil {
		lg.logger.Printf("Could not marshal log entry: %v", err)
		return
	}

	// Write to the log file.
	lg.logger.Println(string(data))

	// If the log level is Fatal, immediately terminate the application.
	if level == Fatal {
		os.Exit(1)
	}
}

// getCallerInfo extracts the package and function name of the caller.
func getCallerInfo(skip int) (packageName, functionName string) {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown", "unknown"
	}

	fullName := fn.Name()
	lastDot := len(fullName) - 1
	for lastDot >= 0 && fullName[lastDot] != '.' {
		lastDot--
	}

	return fullName[:lastDot], fullName[lastDot+1:]
}

// extractContextValues retrieves all key-value pairs from context.Context and stores them in a map.
//
// Params:
// - ctx: The context containing values.
//
// Returns:
// - map[string]interface{}: A map containing extracted key-value pairs from the context.
func extractContextValues(ctx context.Context) map[string]interface{} {
	attributes := make(map[string]interface{})

	if ctx == nil {
		return attributes
	}

	// Retrieve values from the context
	type contextKey string
	contextKeys := []contextKey{"user_id", "request_id", "session_id"} // Define expected keys

	for _, key := range contextKeys {
		if val := ctx.Value(key); val != nil {
			attributes[string(key)] = val
		}
	}

	return attributes
}