package applogger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestLog(t *testing.T) {
	directoryPath := "./tmp"
	filePath := directoryPath + "/log.ndjson"
	os.MkdirAll(directoryPath, os.ModePerm)

	logger := AppLogger{Path: filePath}
	logger.Initialise()

	logger.Log("INFO", "main", "app", "This is a test")
	logger.Log("ERROR", "main", "app", "Cannot open file")
	logger.LogHTTP("INFO", "controller", "perform", "Response: { \"status\": 200}", 200, 0.74453)
	logger.LogHTTP("ERROR", "controller", "perform", "Response: { \"status\": 500}", 500, 2.73353)

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		errNDJSON := isJSON(fmt.Sprintf("%v", scanner.Text()))

		if errNDJSON != nil {
			os.RemoveAll(directoryPath)
			t.Fatalf("line is not in a json format %s with error %s", scanner.Text(), errNDJSON)
		}
	}
	os.RemoveAll(directoryPath)

}

func isJSON(s string) error {
	var js map[string]interface{}
	err := json.Unmarshal([]byte(s), &js)
	return err
}
