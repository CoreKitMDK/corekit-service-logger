package logger

import (
	"encoding/json"
	"fmt"
)

// Console Logging.Console implements the ILogger interface
type Console struct {
	minLogLevel Level
}

func (lc *Console) LogMessage(level Level, message LogMessage) error {
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return lc.Log(level, string(jsonBytes))
}

// NewLoggerConsole creates a new instance of LoggerConsole
func NewLoggerConsole(minLogLevel Level) *Console {
	return &Console{
		minLogLevel: minLogLevel,
	}
}

// Log writes the logger message to the console
func (lc *Console) Log(level Level, message string) error {
	fmt.Printf("%s\n", message)
	return nil
}

// ShouldLogLevel checks if the given logger level meets the minimum logger level
func (lc *Console) ShouldLogLevel(level Level) bool {
	return level >= lc.minLogLevel
}
