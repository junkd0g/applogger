package applogger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofrs/uuid"
)

var (
	generalLogger *log.Logger
	errorLogger   *log.Logger
)

type AppLogger struct {
	Path string
}

type AppLoggerInterface interface {
	Log(level string, logPackage string, logFunc string, message string)
	LogHTTP(level string, logPackage string, logFunc string, message string, code int, duration float64)
}

// logNDJOSNHTTP json format for logs in lib and controller packages
type logNDJOSN struct {
	PID        string    `json:"pid"`
	Level      string    `json:"level"`
	LogPackage string    `json:"package"`
	LogFunc    string    `json:"func"`
	Message    string    `json:"message"`
	DOB        time.Time `json:"time"`
}

// logNDJOSNHTTP json format for logs in the main package
type logNDJOSNHTTP struct {
	PID        string    `json:"pid"`
	Level      string    `json:"level"`
	LogPackage string    `json:"package"`
	LogFunc    string    `json:"func"`
	Message    string    `json:"message"`
	DOB        time.Time `json:"time"`
	Code       int       `json:"code"`
	Duration   float64   `json:"duration"`
}

func (r AppLogger) Initialise() {
	generalLog, err := os.OpenFile(r.Path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	generalLogger = log.New(generalLog, "", 0)
	errorLogger = log.New(generalLog, "", 0)
}

// Log writting to a ndjson file logs for lib and controller packages
func (r AppLogger) Log(level string, logPackage string, logFunc string, message string) {

	s1 := time.Now()
	u := uuid.Must(uuid.NewV4())

	x := logNDJOSN{PID: u.String(), Level: level, LogPackage: logPackage, LogFunc: logFunc, Message: message, DOB: s1}
	res2B, _ := json.Marshal(x)
	generalLogger.Println(string(res2B))
}

// LogHTTP writting to a ndjson file logs for the main package
// the difference is that we are recording the http status
// and the duration of the request
func (r AppLogger) LogHTTP(level string, logPackage string, logFunc string, message string, code int, duration float64) {

	s1 := time.Now()
	u := uuid.Must(uuid.NewV4())

	x := logNDJOSNHTTP{PID: u.String(), Level: level, LogPackage: logPackage, LogFunc: logFunc, Message: message, DOB: s1, Code: code, Duration: duration}
	res2B, _ := json.Marshal(x)
	generalLogger.Println(string(res2B))
}
