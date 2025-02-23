# Applogger - The Structured Logging Library for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/junkd0g/applogger)](https://goreportcard.com/report/github.com/junkd0g/applogger)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://pkg.go.dev/badge/github.com/junkd0g/applogger.svg)](https://pkg.go.dev/github.com/junkd0g/applogger)

## 🚀 Overview

`applogger` is a **structured logging library** for Go that writes logs in **NDJSON format** (Newline Delimited JSON).

- **Structured Logging**: Outputs logs in JSON format for easy parsing.
- **Log Levels**: Supports Debug, Info, Warn, Error, and Fatal.
- **Concurrency Safe**: Uses mutex locking for safe concurrent writes.
- **Automatic Caller Info**: Captures the package and function where the log originated.
- **HTTP Logging**: Supports logging HTTP response codes and request durations.
- **Context-Aware Logging**: Supports extracting key-value pairs from `context.Context`.
- **Graceful Shutdown**: Ensures log files are properly closed on termination.

---

## 📦 Installation

To install `applogger`, simply run:

```sh
go get -u github.com/junkd0g/applogger
```

## 🚀 Usage

Basic Logging

```go
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

```

Logging with Context (Key-Value Pair Extraction)

```go
package main

import (
	"context"
	"github.com/junkd0g/applogger"
)

func main() {
	logger, _ := applogger.NewLogger("app.log")
	defer logger.Close()

	ctx := context.WithValue(context.Background(), "userID", "12345")
	logger.Log(ctx, applogger.Info, "User logged in")
}
```

HTTP Logging

```go
logger.LogHTTP(context.Background(), applogger.Info, "GET /api/user successful", 200, 0.125)
```

Fatal Logging (Exits Application)

```go
logger.Log(context.Background(), applogger.Fatal, "Critical system failure!")
```

## 📝 License

This project is licensed under the MIT License. See the LICENSE file for details.

## Authors

- **Iordanis Paschalidis** -[junkd0g](https://github.com/junkd0g)
