package tests

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CoreKitMDK/corekit-service-logger/v2/pkg/logger"
)

func TestLoggerConfiguration(t *testing.T) {

	config := logger.NewConfiguration()

	config.UseConsole = true

	config.UseNATS = true
	config.NatsURL = "nats://localhost:4222"

	config.NatsPassword = "internal-logger-broker"
	config.NatsUsername = "internal-logger-broker"

	marshal, err := json.Marshal(config)
	if err != nil {
		return
	}
	println(string(marshal))

	ogger := config.Init()
	defer ogger.Stop()

	ogger.Log(logger.INFO, "Test message")

	ogger.Logf(logger.INFO, "Test message %s", "with params")
	ogger.LogContext(logger.INFO, nil, "Test message", "with keys")
	ogger.LogJson(logger.INFO, map[string]string{"key1": "value1", "key2": "value2"})

	time.Sleep(2 * time.Second)
}
