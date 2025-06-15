package logger

import (
	"encoding/json"
	"fmt"
	"time"
)

type Fallback struct {
}

func NewLoggerFallback() *Fallback {
	return &Fallback{}
}

func (lf *Fallback) LogMessage(level Level, message LogMessage) error {
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return lf.Log(level, string(jsonBytes))
}

func (lf *Fallback) Log(level Level, message string) error {
	fmt.Printf("%s - [%s] : %s\n", time.Now().Format("2006-01-02 15:04:05"), LogLevelToString(level), message)
	return nil
}

func (lf *Fallback) ShouldLogLevel(level Level) bool {
	return true
}
