# applogger

[![Go Report Card](https://goreportcard.com/badge/github.com/junkd0g/applogger)](https://goreportcard.com/report/github.com/junkd0g/applogger)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://pkg.go.dev/badge/github.com/junkd0g/applogger.svg)](https://pkg.go.dev/github.com/junkd0g/applogger)

`applogger` is a structured logging library for Go that writes logs in NDJSON
(Newline-Delimited JSON) format to both a file and stdout.

## Why

Most Go services need structured logs that are cheap to grep, ship, and parse.
`applogger` keeps that path small: one logger, one file handle, one mutex, and
JSON entries that include the calling package and function automatically — no
zap, no zerolog, no extra deps.

## Features

- Structured NDJSON output (one JSON object per line).
- Five log levels: `Debug`, `Info`, `Warn`, `Error`, `Fatal`.
- Concurrency-safe writes via internal mutex.
- Automatic caller info (package + function).
- HTTP helper that records status code and request duration.
- Context-aware: extra fields can travel on `context.Context`.
- Per-logger default fields via `WithFields`.
- Writes to both stdout and a file with a single `io.MultiWriter`.

## Requirements

- Go 1.23 or newer.
- No external dependencies.

## Installation

```sh
go get -u github.com/junkd0g/applogger
```

## Quick start

```go
package main

import (
    "context"
    "log"

    "github.com/junkd0g/applogger"
)

func main() {
    logger, err := applogger.NewLogger("app.log")
    if err != nil {
        log.Fatalf("init logger: %v", err)
    }
    defer logger.Close()

    logger.Log(context.Background(), applogger.Info, "service started")
}
```

## Usage

### Default fields

`WithFields` returns a new logger that attaches the given key/value pairs to
every entry it writes:

```go
logger = logger.WithFields(map[string]interface{}{
    "service": "payment",
    "version": "1.0.3",
})
```

### Context fields

Attach a `map[string]interface{}` under `applogger.ApploggerFieldsKey` and the
logger will merge it into each entry's `attributes`:

```go
ctx := context.WithValue(context.Background(),
    applogger.ApploggerFieldsKey,
    map[string]interface{}{
        "request_id": "req-001",
        "user_id":    "12345",
    },
)
logger.Log(ctx, applogger.Info, "request received")
```

### HTTP logging

```go
logger.LogHTTP(ctx, applogger.Info, "GET /api/user", 200, 0.125)
```

### Fatal

`Fatal` writes the entry, then calls `os.Exit(1)`:

```go
logger.Log(ctx, applogger.Fatal, "unrecoverable failure")
```

## API Reference

### Types

```go
type LogLevel int

const (
    Debug LogLevel = iota
    Info
    Warn
    Error
    Fatal
)

type LogEntry struct {
    PID        string                 `json:"pid"`
    Level      string                 `json:"level"`
    Package    string                 `json:"package"`
    Func       string                 `json:"func"`
    Message    string                 `json:"message"`
    Timestamp  time.Time              `json:"timestamp"`
    Code       int                    `json:"code,omitempty"`
    Duration   float64                `json:"duration,omitempty"`
    Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type Logger struct { /* unexported fields */ }
```

### Functions and methods

```go
func NewLogger(path string) (*Logger, error)

func (lg *Logger) Close() error
func (lg *Logger) WithFields(fields map[string]interface{}) *Logger
func (lg *Logger) Log(ctx context.Context, level LogLevel, message string)
func (lg *Logger) LogHTTP(ctx context.Context, level LogLevel, message string, code int, duration float64)
```

### Context key

```go
type ContextKey string
const ApploggerFieldsKey ContextKey = "applogger_fields"
```

Store a `map[string]interface{}` under this key on a `context.Context` to have
its entries merged into the log entry's `attributes` field.

## Testing

```sh
go test ./...
```

With coverage:

```sh
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Contributing

1. Fork the repository.
2. Create a feature branch (`git checkout -b feature/your-feature`).
3. Commit your changes.
4. Push to the branch (`git push origin feature/your-feature`).
5. Open a pull request.

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE)
file for details.

## Author

Iordanis Paschalidis — [@junkd0g](https://github.com/junkd0g)
