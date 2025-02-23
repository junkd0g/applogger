package main

import (
	"context"
	"log"
	"time"

	"github.com/junkd0g/applogger"
)

func main() {
	logger, err := applogger.NewLogger("app.log")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	// Add extra default fields.
	logger = logger.WithFields(map[string]interface{}{
		"service": "payment",
		"version": "1.0.3",
	})

	// Create a context with values.
	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", "12345")
	ctx = context.WithValue(ctx, "request_id", "req-001")
	ctx = context.WithValue(ctx, "session_id", "sess-789")

	// Log a simple message.
	logger.Log(ctx, applogger.Info, "Service started successfully")

	// Log an HTTP-related event.
	logger.LogHTTP(ctx, applogger.Debug, "HTTP response received", 200, 0.456)

	// Wait a moment to see different timestamps.
	time.Sleep(1 * time.Second)

	// Log an error.
	logger.Log(ctx, applogger.Error, "Encountered an unexpected error")
}
