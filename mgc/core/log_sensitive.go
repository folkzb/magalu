package core

import (
	"encoding/json"
	"fmt"
	"os"
)

type LogSensitive string

const logSensitiveEnvVar = "MGC_SDK_LOG_SENSITIVE"

var shouldLogSensitveStatus *bool

func shouldLogSensitive() bool {
	if shouldLogSensitveStatus == nil {
		b := os.Getenv(logSensitiveEnvVar) == "1"
		shouldLogSensitveStatus = &b
	}
	return *shouldLogSensitveStatus
}

func (s LogSensitive) MarshalJSON() ([]byte, error) {
	var text string
	if shouldLogSensitive() {
		text = string(s)
	} else {
		text = fmt.Sprintf("[REDACTED %d CHARS]", len(s))
	}

	return json.Marshal(text)
}
