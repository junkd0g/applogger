# Applogger

Package junkd0g/applogger is a simple ndjson logger

## Installing

go get -u github.com/junkd0g/applogger

## Running the tests

go test ./...

## Example

```go
package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/junkd0g/applogger"
)

func HelloWorld(w http.ResponseWriter, r *http.Request) {

	logger := applogger.AppLogger{Path: "/tmp/logger.ndjson"}
	logger.Initialise()

	response := "{ \"message\" : \"Hello World\"}"
	logger.Log("INFO", "main", "HelloWorld", response)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(response))
}

func main() {

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", HelloWorld)

	http.ListenAndServe(":8076", router)
}

```

## Authors

* **Iordanis Paschalidis** -[PurpleBooth](https://github.com/junkd0g)