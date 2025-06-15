package tests

import (
	"corekitmdk.com/logger/v2/pkg/logger"
	"testing"
	"time"
)

func TestLoggerNats(t *testing.T) {
	natsLogger, err := logger.NewLoggerNATSWithAuth("", "internal-logger-broker", "internal-logger-broker", logger.DEBUG)
	if err != nil {
		t.Error(err)
	}
	//defer natsLogger.Close()

	//natsLogger.Log(logger.DEBUG, "Test message")
	ogger := logger.NewLogger(100, natsLogger)
	//defer ogger.Stop()

	ogger.Log(logger.INFO, "Test message")

	ogger.Logf(logger.INFO, "Test message %s", "with params")
	ogger.LogContext(logger.INFO, nil, "Test message", "with keys")
	ogger.LogJson(logger.INFO, map[string]string{"key1": "value1", "key2": "value2"})

	time.Sleep(2 * time.Second)
}
