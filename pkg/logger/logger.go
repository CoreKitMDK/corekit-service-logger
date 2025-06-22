package logger

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/CoreKitMDK/corekit-service-logger/v2/internal/logger"
)

var Logger IMultiLogger = NewLogger(100, NewLoggerConsole(DEBUG))
var loggerFallback ILogger = NewLoggerFallback()

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
	UNKNOWN
)

// LogMessage represents the structure of a log message sent to NATS
type LogMessage struct {
	Timestamp string            `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Tags      map[string]string `json:"tags"`
}

func LogLevelToString(l Level) string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func isValidLogLevel(level Level) bool {
	return level >= DEBUG && level <= UNKNOWN
}

type IMultiLogger interface {
	Log(level Level, args ...interface{})
	Logf(level Level, format string, args ...interface{})
	LogJson(level Level, args ...interface{})
	LogContext(level Level, context context.Context, keys ...interface{})
	Stop()
}

type ILogger interface {
	Log(level Level, message string) error
	LogMessage(level Level, message LogMessage) error
	ShouldLogLevel(level Level) bool
}

type logEntry struct {
	level   Level
	message string
}

type MultiLogger struct {
	loggers   []ILogger
	bufferLen int
	tags      map[string]string
	logCh     chan logEntry
	quitLogCh chan struct{}
	stopped   bool
	metrics   Metrics
}

func (l *MultiLogger) processLog(entry logEntry) {
	start := time.Now()

	var didLog = false

	for _, logger := range l.loggers {
		if logger.ShouldLogLevel(entry.level) {

			logMsg := LogMessage{
				Timestamp: time.Now().Format(time.RFC3339),
				Level:     LogLevelToString(entry.level),
				Message:   entry.message,
				Tags:      l.tags,
			}

			if err := logger.LogMessage(entry.level, logMsg); err != nil {
				fallbackLog(entry.level, fmt.Sprintln("Error logging message: ", err))
				l.metrics.LoggerFailed()
			} else {
				didLog = true
			}
		}
	}

	if !didLog {
		fallbackLog(entry.level, entry.message)
	}

	l.metrics.ChMessageProcessingTimeMsAvgAdd(time.Since(start).Milliseconds())
}

func (l *MultiLogger) log(level Level, message string) {

	l.metrics.ChTotalMessagesInc()
	l.metrics.ChCurrentUsageSet(len(l.logCh))

	if float32(len(l.logCh))/float32(l.bufferLen) > 0.8 {
		if level == DEBUG || level == INFO || level == WARN {
			// Skip verbose logging for low-priority messages
			//fallbackLog(level, " [OVERFLOW] Channel near full capacity ignoring low priority message: "+message)
			l.metrics.ChDroppedMessagesInc()
			return
		}
	}

	select {
	case l.logCh <- logEntry{level, message}:
		l.metrics.ChProcessedMessagesInc()
	default:
		fallbackLog(level, "Channel overflow detected: "+message)
		if level == ERROR || level == FATAL {
			go func() {
				l.processLog(logEntry{level, message})
			}()
		} else {
			fallbackLog(level, " [OVERFLOW] Channel overflowed ignoring low priority message: "+message)
			l.metrics.ChDroppedMessagesInc()
		}
	}
}

func (l *MultiLogger) formatTags() string {
	if len(l.tags) == 0 {
		return ""
	}

	var builder strings.Builder

	for key, value := range l.tags {
		builder.WriteString(key)
		builder.WriteString(":")
		builder.WriteString(value)
		builder.WriteString(",")
	}

	result := builder.String()
	if len(result) > 0 {
		result = result[:len(result)-1]
	}

	result += ";"

	return result
}

func (l *MultiLogger) startWorker() {
	defer close(l.logCh)

	for {
		select {
		case entry := <-l.logCh:
			l.processLog(entry)
		case <-l.quitLogCh:
			for entry := range l.logCh {
				l.processLog(entry)
			}
			return
		}
	}
}

func NewLogger(bufferLen int, loggers ...ILogger) *MultiLogger {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("Error getting hostname: %v\n", err)
		hostname = "unknown"
	}

	tags := make(map[string]string)
	tags["hostname"] = hostname

	logger := &MultiLogger{
		loggers:   loggers,
		bufferLen: bufferLen,
		logCh:     make(chan logEntry, bufferLen*10),
		quitLogCh: make(chan struct{}),
		stopped:   false,
		tags:      tags,
		metrics: Metrics{
			AliveSince:                   time.Now(),
			ChCurrentUsage:               0,
			ChPeakUsage:                  0,
			ChDroppedMessages:            0,
			ChProcessedMessages:          0,
			ChTotalMessages:              0,
			ChMessageProcessingTimeMsAvg: 0,
			ChMessageProcessingTimeMsMax: 0,
			LoggerFailedCount:            0,
			LastLoggerFailed:             time.Now(),
			DebugCount:                   0,
			InfoCount:                    0,
			WarnCount:                    0,
			ErrorCount:                   0,
			FatalCount:                   0,
			UnknownCount:                 0,
		},
	}
	go logger.startWorker()
	return logger
}

func (l *MultiLogger) Stop() {
	l.stopped = true
	close(l.quitLogCh)
}

func (l *MultiLogger) Log(level Level, args ...interface{}) {
	if len(args) == 0 {
		return
	}

	if l.stopped == true {
		fallbackLog(ERROR, fmt.Sprintln("Error logging message: ", "logger is stopped ", level))
		return
	}

	if !isValidLogLevel(level) {
		fallbackLog(ERROR, fmt.Sprintln("Error logging message: ", "invalid logger level ", level))
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s - [%s] : ", timestamp, LogLevelToString(level)))

	builder.WriteString(logger.Stringify(args))

	if level == FATAL || level == ERROR {
		buf := make([]byte, 1<<16)
		bufLen := runtime.Stack(buf, true)
		builder.WriteString("\n Stack trace : \n")
		builder.WriteString(string(buf[:bufLen]))
	}

	builder.WriteString("\n")

	l.log(level, builder.String())
}

func (l *MultiLogger) Logf(level Level, format string, args ...interface{}) {
	if len(args) == 0 {
		return
	}

	if l.stopped == true {
		fallbackLog(ERROR, fmt.Sprintln("Error logging message: ", "logger is stopped ", level))
		return
	}

	if !isValidLogLevel(level) {
		fallbackLog(ERROR, fmt.Sprintln("Error logging message: ", "invalid logger level ", level))
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formattedMessage := fmt.Sprintf("%s - [%s] : ", timestamp, LogLevelToString(level)) + fmt.Sprintf(format, args...)
	l.log(level, formattedMessage)
}

func (l *MultiLogger) LogJson(level Level, args ...interface{}) {
	//jsonData, err := json.Marshal(args)
	//if err != nil {
	//	fallbackLog(ERROR, "Failed to serialize logger to JSON: "+err.Error())
	//	return
	//}
	//timestamp := time.Now().Format("2006-01-02 15:04:05")
	//l.logger(level, fmt.Sprintf("%s - [%s] : %s", timestamp, LogLevelToString(level), string(jsonData)))
	l.Log(level, args...)
}

func (l *MultiLogger) LogContext(level Level, ctx context.Context, keys ...interface{}) {
	if l.stopped {
		fallbackLog(ERROR, fmt.Sprintln("Error logging message: ", "logger is stopped ", level))
		return
	}

	if !isValidLogLevel(level) {
		fallbackLog(ERROR, fmt.Sprintln("Error logging message: ", "invalid logger level ", level))
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	contextData := extractKnownContextKeys(ctx, keys)

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s - [%s] : ", timestamp, LogLevelToString(level)))
	if len(contextData) > 0 {
		builder.WriteString("Context: [")
		for key, value := range contextData {
			builder.WriteString(fmt.Sprintf("%s=%v ", key, value))
		}
		builder.WriteString("] ")
	}
	builder.WriteString("\n")

	if level == FATAL || level == ERROR {
		buf := make([]byte, 1<<16)
		bufLen := runtime.Stack(buf, true)
		builder.WriteString("\n Stack trace : \n")
		builder.WriteString(string(buf[:bufLen]))
	}

	l.log(level, builder.String())
}

func fallbackLog(level Level, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	err := loggerFallback.Log(level, fmt.Sprintf("%s - [FALLBACK] [%s] : %s\n", timestamp, LogLevelToString(level), message))
	if err != nil {
		return
	}
}

func extractKnownContextKeys(ctx context.Context, keys ...interface{}) map[interface{}]interface{} {
	result := make(map[interface{}]interface{})
	for _, key := range keys {
		if value := ctx.Value(key); value != nil {
			result[key] = value
		}
	}
	return result
}
