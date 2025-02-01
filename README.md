# Applogger - The Structured Logging Library for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/junkd0g/applogger)](https://goreportcard.com/report/github.com/junkd0g/applogger)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://pkg.go.dev/badge/github.com/your-repo/applogger.svg)](https://pkg.go.dev/github.com/junkd0g/applogger)

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
go get -u github.com/your-repo/applogger

## Authors

* **Iordanis Paschalidis** -[junkd0g](https://github.com/junkd0g)
```
