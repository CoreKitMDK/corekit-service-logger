package logger

import (
	"sync"
	"time"
)

type Metrics struct {
	AliveSince time.Time

	ChCurrentUsage int64
	ChPeakUsage    int64

	ChDroppedMessages   int64
	ChProcessedMessages int64
	ChTotalMessages     int64

	ChMessageProcessingTimeMsAvg int64 // not valid until 100 messages processed
	ChMessageProcessingTimeMsMax int64

	LoggerFailedCount int64
	LastLoggerFailed  time.Time

	DebugCount   int64
	InfoCount    int64
	WarnCount    int64
	ErrorCount   int64
	FatalCount   int64
	UnknownCount int64

	mutex sync.Mutex
}

func (m *Metrics) LevelCountInc(level Level) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch level {
	case DEBUG:
		m.DebugCount += 1
	case INFO:
		m.InfoCount += 1
	case WARN:
		m.WarnCount += 1
	case ERROR:
		m.ErrorCount += 1
	case FATAL:
		m.FatalCount += 1
	default:
		m.UnknownCount += 1
	}
}

func (m *Metrics) LoggerFailed() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.LoggerFailedCount += 1
	m.LastLoggerFailed = time.Now()
}

func (m *Metrics) ChCurrentUsageSet(count int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.ChPeakUsage < int64(count) {
		m.ChPeakUsage = int64(count)
	}
	m.ChCurrentUsage = int64(count)
}

func (m *Metrics) ChDroppedMessagesInc() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.ChDroppedMessages += 1
}

func (m *Metrics) ChProcessedMessagesInc() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.ChProcessedMessages += 1
}

func (m *Metrics) ChTotalMessagesInc() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.ChTotalMessages += 1
}

func (m *Metrics) ChMessageProcessingTimeMsAvgAdd(processingTimeMs int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.ChMessageProcessingTimeMsMax < processingTimeMs {
		m.ChMessageProcessingTimeMsMax = processingTimeMs
	}
	m.ChMessageProcessingTimeMsAvg = (m.ChMessageProcessingTimeMsAvg*99 + processingTimeMs) / 100
}
