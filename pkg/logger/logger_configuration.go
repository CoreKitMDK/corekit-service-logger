package logger

import "encoding/json"

// Logger should init itself from a json configuration
type Configuration struct {
	UseConsole   bool   `json:"use_console"`
	UseNATS      bool   `json:"use_nats"`
	NatsURL      string `json:"nats_url"`
	NatsUsername string `json:"nats_username"`
	NatsPassword string `json:"nats_password"`
}

func NewConfiguration() *Configuration {
	return &Configuration{}
}

func FromJsonString(jsonString string) (*Configuration, error) {
	var config Configuration
	err := json.Unmarshal([]byte(jsonString), &config) // Note the & operator here
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *Configuration) Init() *MultiLogger {
	var loggers []ILogger

	if c.UseConsole {
		consoleLogger := NewLoggerConsole(Level(0))
		loggers = append(loggers, consoleLogger)
	}

	if c.UseNATS {
		if c.NatsUsername != "" && c.NatsPassword != "" {
			if natsLogger, err := NewLoggerNATSWithAuth(c.NatsURL, c.NatsUsername, c.NatsPassword, Level(0)); err == nil {
				loggers = append(loggers, natsLogger)
			}
		} else {
			if natsLogger, err := NewLoggerNATS(c.NatsURL, Level(0)); err == nil {
				loggers = append(loggers, natsLogger)
			}
		}
	}

	multiLogger := NewLogger(100, loggers...)
	return multiLogger
}
