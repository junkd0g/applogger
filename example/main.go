package main

import (
	"context"
	"log"
	"time"

	"github.com/junkd0g/applogger"
)

func main() {
	// Initialize the logger to write to "app.log"
	logger, err := applogger.NewLogger("app.log")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	// Example 1: Log using context extra fields.
	// Create a context and store arbitrary fields under "applogger_fields".
	extraFields := map[string]interface{}{
		"user_id":    "user-001",
		"session_id": "sess-abc",
		"custom":     "extra info",
	}
	ctx := context.WithValue(context.Background(), "applogger_fields", extraFields)
	logger.Log(ctx, applogger.Info, "Logging with context extra fields")

	// Example 2: Log using WithFields to attach default fields to the logger.
	loggerWithDefaults := logger.WithFields(map[string]interface{}{
		"service": "myservice",
		"version": "2.0",
	})
	loggerWithDefaults.Log(context.Background(), applogger.Debug, "Logging with default fields from WithFields")

	// Example 3: Log an HTTP event.
	logger.LogHTTP(context.Background(), applogger.Warn, "HTTP event occurred", 404, 0.567)

	// Wait a moment so timestamps differ.
	time.Sleep(500 * time.Millisecond)
}
