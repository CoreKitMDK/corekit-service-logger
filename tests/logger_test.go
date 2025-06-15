package tests

import (
	"bytes"
	"context"
	"corekitmdk.com/logger/v2/pkg/logger"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// MockLogger implements the logger.ILogger interface for testing
type MockLogger struct {
	buffer       bytes.Buffer
	minLogLevel  logger.Level
	shouldFail   bool
	loggedCalls  int
	levelChecked logger.Level
}

func NewMockLogger(minLogLevel logger.Level) *MockLogger {
	return &MockLogger{
		minLogLevel: minLogLevel,
		shouldFail:  false,
	}
}

func (ml *MockLogger) LogMessage(level logger.Level, message logger.LogMessage) error {
	return nil
}

func (ml *MockLogger) Log(level logger.Level, message string) error {
	ml.loggedCalls++
	if ml.shouldFail {
		return fmt.Errorf("mock logger failed intentionally")
	}
	ml.buffer.WriteString(message)
	return nil
}

func (ml *MockLogger) ShouldLogLevel(level logger.Level) bool {
	ml.levelChecked = level
	return level >= ml.minLogLevel
}

func (ml *MockLogger) GetLoggedContent() string {
	return ml.buffer.String()
}

func (ml *MockLogger) ResetBuffer() {
	ml.buffer.Reset()
	ml.loggedCalls = 0
}

func TestLoggerConsole(t *testing.T) {
	// Redirect stdout to capture console output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a console logger with minimum level of INFO
	consoleLogger := logger.NewLoggerConsole(logger.INFO)

	// Test ShouldLogLevel method
	if consoleLogger.ShouldLogLevel(logger.DEBUG) {
		t.Error("DEBUG level should not be logged with INFO minimum level")
	}
	if !consoleLogger.ShouldLogLevel(logger.INFO) {
		t.Error("INFO level should be logged with INFO minimum level")
	}
	if !consoleLogger.ShouldLogLevel(logger.ERROR) {
		t.Error("ERROR level should be logged with INFO minimum level")
	}

	// Test Log method
	testMessage := "Test console message"
	err := consoleLogger.Log(logger.INFO, testMessage)
	if err != nil {
		t.Errorf("Console logger returned error: %v", err)
	}

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify output
	output := buf.String()
	if !strings.Contains(output, testMessage) {
		t.Errorf("Expected output to contain '%s', got '%s'", testMessage, output)
	}
}

func TestLoggerFallback(t *testing.T) {
	// Redirect stdout to capture console output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create a fallback logger
	fallbackLogger := logger.NewLoggerFallback()

	// Test ShouldLogLevel method
	if !fallbackLogger.ShouldLogLevel(logger.DEBUG) {
		t.Error("Fallback logger should log all levels")
	}
	if !fallbackLogger.ShouldLogLevel(logger.ERROR) {
		t.Error("Fallback logger should log all levels")
	}

	// Test Log method
	testMessage := "Test fallback message"
	err := fallbackLogger.Log(logger.INFO, testMessage)
	if err != nil {
		t.Errorf("Fallback logger returned error: %v", err)
	}

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify output
	output := buf.String()
	if !strings.Contains(output, testMessage) {
		t.Errorf("Expected output to contain '%s', got '%s'", testMessage, output)
	}
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected output to contain log level 'INFO', got '%s'", output)
	}
}

func TestMultiLogger(t *testing.T) {
	// Create mock loggers
	mockDebug := NewMockLogger(logger.DEBUG)
	mockInfo := NewMockLogger(logger.INFO)
	mockError := NewMockLogger(logger.ERROR)

	// Create multi-logger with these mocks
	multiLogger := logger.NewLogger(10, mockDebug, mockInfo, mockError)
	defer multiLogger.Stop()

	// Test Log method with different levels
	multiLogger.Log(logger.DEBUG, "Debug message")
	time.Sleep(10 * time.Millisecond) // Give time for async processing

	if !strings.Contains(mockDebug.GetLoggedContent(), "DEBUG") {
		t.Error("DEBUG message should be logged to debug logger")
	}
	if strings.Contains(mockError.GetLoggedContent(), "DEBUG") {
		t.Error("DEBUG message should not be logged to error logger")
	}

	mockDebug.ResetBuffer()
	mockInfo.ResetBuffer()
	mockError.ResetBuffer()

	// Test ERROR level (should go to all loggers with appropriate level)
	multiLogger.Log(logger.ERROR, "Error message")
	time.Sleep(10 * time.Millisecond) // Give time for async processing

	if !strings.Contains(mockDebug.GetLoggedContent(), "ERROR") {
		t.Error("ERROR message should be logged to debug logger")
	}
	if !strings.Contains(mockInfo.GetLoggedContent(), "ERROR") {
		t.Error("ERROR message should be logged to info logger")
	}
	if !strings.Contains(mockError.GetLoggedContent(), "ERROR") {
		t.Error("ERROR message should be logged to error logger")
	}

	// Test Logf method
	mockDebug.ResetBuffer()
	multiLogger.Logf(logger.INFO, "Formatted %s with %d params", "message", 2)
	time.Sleep(10 * time.Millisecond)

	if !strings.Contains(mockDebug.GetLoggedContent(), "Formatted message with 2 params") {
		t.Error("Formatted message was not logged correctly")
	}

	// Test context logging
	mockDebug.ResetBuffer()
	ctx := context.WithValue(context.Background(), "key1", "value1")
	multiLogger.LogContext(logger.INFO, ctx, "key1")
	time.Sleep(10 * time.Millisecond)

	print(mockDebug.GetLoggedContent())
	if !strings.Contains(mockDebug.GetLoggedContent(), "key1=value1") {
		t.Error("Context value was not logged correctly")
	}
}

func TestLoggerFallbackScenario(t *testing.T) {
	// Create a mock logger that will fail
	mockFailing := NewMockLogger(logger.DEBUG)
	mockFailing.shouldFail = true

	// Redirect stdout to capture fallback output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create multi-logger with failing mock
	multiLogger := logger.NewLogger(10, mockFailing)
	defer multiLogger.Stop()

	// Log a message (should trigger fallback)
	multiLogger.Log(logger.ERROR, "This should go to fallback")
	time.Sleep(10 * time.Millisecond) // Give time for async processing

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify fallback output
	output := buf.String()
	if !strings.Contains(output, "FALLBACK") {
		t.Error("Fallback logger should have been used")
	}
}

func TestLogLevelToString(t *testing.T) {
	tests := []struct {
		level    logger.Level
		expected string
	}{
		{logger.DEBUG, "DEBUG"},
		{logger.INFO, "INFO"},
		{logger.WARN, "WARN"},
		{logger.ERROR, "ERROR"},
		{logger.FATAL, "FATAL"},
		{logger.Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		result := logger.LogLevelToString(tt.level)
		if result != tt.expected {
			t.Errorf("LogLevelToString(%v) = %v, expected %v", tt.level, result, tt.expected)
		}
	}
}

func main() {
	// This function can be used to run the tests directly
	testing.Main(func(pat, str string) (bool, error) { return true, nil },
		[]testing.InternalTest{
			{"TestLoggerConsole", TestLoggerConsole},
			{"TestLoggerFallback", TestLoggerFallback},
			{"TestMultiLogger", TestMultiLogger},
			{"TestLoggerFallbackScenario", TestLoggerFallbackScenario},
			{"TestLogLevelToString", TestLogLevelToString},
		},
		[]testing.InternalBenchmark{},
		[]testing.InternalExample{})
}
