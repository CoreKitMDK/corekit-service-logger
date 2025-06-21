package logger

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// NATS Logging.NATS implements the ILogger interface
type NATS struct {
	minLogLevel Level
	conn        *nats.Conn
	subject     string
	clientID    string
}

// NATSOption is a functional option for configuring the NATS logger
type NATSOption func(*NATS)

// WithClientID sets the client ID for the NATS logger
func WithClientID(clientID string) NATSOption {
	return func(n *NATS) {
		n.clientID = clientID
	}
}

// WithSubject sets the subject for publishing log messages
func WithSubject(subject string) NATSOption {
	return func(n *NATS) {
		n.subject = subject
	}
}

// WithCredentials sets username and password for NATS authentication
func WithCredentials(username, password string) NATSOption {
	return func(n *NATS) {
		// This option doesn't modify the NATS struct directly
		// Instead, it's used when establishing the connection
	}
}

func NewLoggerNATS(url string, minLogLevel Level, options ...NATSOption) (*NATS, error) {
	logger := &NATS{
		minLogLevel: minLogLevel,
		subject:     "logs",                   // Default subject
		clientID:    "internal-logger-broker", // Default client ID
	}

	for _, opt := range options {
		opt(logger)
	}

	natsOpts := []nats.Option{
		nats.Name(logger.clientID),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(10),
	}

	nc, err := nats.Connect(url, natsOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server: %w", err)
	}

	logger.conn = nc
	return logger, nil
}

func NewLoggerNATSWithAuth(url string, username, password string, minLogLevel Level, options ...NATSOption) (*NATS, error) {
	logger := &NATS{
		minLogLevel: minLogLevel,
		subject:     "logs",                   // Default subject
		clientID:    "internal-logger-broker", // Default client ID
	}

	for _, opt := range options {
		opt(logger)
	}

	nc, err := nats.Connect(url,
		nats.Name(logger.clientID),
		nats.UserInfo(username, password),
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(10),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server: %w", err)
	}

	logger.conn = nc
	return logger, nil
}

func (ln *NATS) LogMessage(level Level, message LogMessage) error {
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return ln.Log(level, string(jsonBytes))
}

func (ln *NATS) Log(level Level, message string) error {
	if ln.conn == nil || ln.conn.IsClosed() {
		return fmt.Errorf("NATS connection is closed or not initialized")
	}

	err := ln.conn.Publish(ln.subject, []byte(message))
	if err != nil {
		return err
	}

	err = ln.conn.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (ln *NATS) ShouldLogLevel(level Level) bool {
	return level >= ln.minLogLevel
}

func (ln *NATS) Close() {
	if ln.conn != nil && !ln.conn.IsClosed() {
		ln.conn.Close()
	}
}
