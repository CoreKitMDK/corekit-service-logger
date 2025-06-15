package logger

// Logger should init itself from a json configuration
type Configuration struct {
	UseConsole   bool
	UseNATS      bool
	NatsURL      string
	NatsUsername string
	NatsPassword string
}

func NewConfiguration() *Configuration {
	return &Configuration{}
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
