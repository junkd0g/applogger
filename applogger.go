package applogger

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents different levels of log severity.
type LogLevel int

const (
	Debug LogLevel = iota // Debug-level messages.
	Info                  // Informational messages.
	Warn                  // Warning messages.
	Error                 // Error messages.
	Fatal                 // Fatal messages that exit the application.
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
	PID        string                 `json:"pid"`                  // Unique identifier for the log event.
	Level      string                 `json:"level"`                // Log severity level.
	Package    string                 `json:"package"`              // Package name where the log was generated.
	Func       string                 `json:"func"`                 // Function name where the log was generated.
	Message    string                 `json:"message"`              // Log message.
	Timestamp  time.Time              `json:"timestamp"`            // Time when the log was created.
	Code       int                    `json:"code,omitempty"`       // HTTP status code (if applicable).
	Duration   float64                `json:"duration,omitempty"`   // Request duration in seconds (if applicable).
	Attributes map[string]interface{} `json:"attributes,omitempty"` // Merged attributes from context and default fields.
}

// Logger is a structured logging system for NDJSON logs.
type Logger struct {
	logger        *log.Logger            // Internal Go logger.
	mu            *sync.Mutex            // Mutex for concurrent safety.
	file          *os.File               // Log file handle.
	defaultFields map[string]interface{} // Extra default fields attached to every log entry.
}

// NewLogger initializes a new Logger instance that writes logs to the specified file and stdout.
func NewLogger(path string) (*Logger, error) {
	// Open or create the log file.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}

	// Create a multiwriter to write logs to both stdout and the file.
	mw := io.MultiWriter(os.Stdout, f)
	l := log.New(mw, "", 0)

	return &Logger{
		logger:        l,
		file:          f,
		mu:            &sync.Mutex{},
		defaultFields: make(map[string]interface{}),
	}, nil
}

// Close properly closes the log file.
func (lg *Logger) Close() error {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	if lg.file != nil {
		return lg.file.Close()
	}
	return nil
}

// WithFields returns a new Logger that automatically includes the provided key/value pairs.
func (lg *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := make(map[string]interface{})
	for k, v := range lg.defaultFields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	return &Logger{
		logger:        lg.logger,
		mu:            lg.mu,
		file:          lg.file,
		defaultFields: newFields,
	}
}

// Log writes a structured log entry to the outputs.
func (lg *Logger) Log(ctx context.Context, level LogLevel, message string) {
	lg.logInternal(ctx, level, message, 0, 0, 3)
}

// LogHTTP logs an HTTP event with status code and duration.
func (lg *Logger) LogHTTP(ctx context.Context, level LogLevel, message string, code int, duration float64) {
	lg.logInternal(ctx, level, message, code, duration, 3)
}

// logInternal is the core logging function.
func (lg *Logger) logInternal(ctx context.Context, level LogLevel, msg string, code int, duration float64, skip int) {
	lg.mu.Lock()
	defer lg.mu.Unlock()

	// Get caller information.
	pkgName, funcName := getCallerInfo(skip)

	// Generate a unique PID based on the current time.
	pid := time.Now().Format("20060102150405")

	// Extract arbitrary extra fields from context (if any).
	ctxFields := extractContextValues(ctx)

	// Merge default fields and context fields.
	attributes := make(map[string]interface{})
	for k, v := range lg.defaultFields {
		attributes[k] = v
	}
	for k, v := range ctxFields {
		attributes[k] = v
	}

	// Create the log entry.
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

	// Serialize the entry to JSON.
	data, err := json.Marshal(entry)
	if err != nil {
		lg.logger.Printf("Could not marshal log entry: %v", err)
		return
	}

	// Write the JSON log entry.
	lg.logger.Println(string(data))

	// Exit if level is Fatal.
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

// extractContextValues retrieves arbitrary key/value pairs from the context.
// It expects that any extra fields are stored in a map[string]interface{}
// under the dedicated key "applogger_fields".
func extractContextValues(ctx context.Context) map[string]interface{} {
	attributes := make(map[string]interface{})
	if ctx == nil {
		return attributes
	}
	if extra, ok := ctx.Value("applogger_fields").(map[string]interface{}); ok {
		for k, v := range extra {
			attributes[k] = v
		}
	}
	return attributes
}
