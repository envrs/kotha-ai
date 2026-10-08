package logging

// Logger is the interface for structured logging.
// It abstracts the global logging singleton and enables dependency injection.
type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	InfoPersist(msg string, args ...any)
	DebugPersist(msg string, args ...any)
	WarnPersist(msg string, args ...any)
	ErrorPersist(msg string, args ...any)
	RecoverPanic(name string, cleanup func())
	AppendToSessionLogFile(sessionId string, filename string, content string) string
	WriteRequestMessageJson(sessionId string, requestSeqId int, message any) string
	WriteRequestMessage(sessionId string, requestSeqId int, message string) string
	AppendToStreamSessionLogJson(sessionId string, requestSeqId int, jsonableChunk any) string
	AppendToStreamSessionLog(sessionId string, requestSeqId int, chunk string) string
	WriteChatResponseJson(sessionId string, requestSeqId int, response any) string
	WriteToolResultsJson(sessionId string, requestSeqId int, toolResults any) string
}

// defaultLogger is the default implementation of Logger that wraps package-level functions.
type defaultLogger struct{}

func (d *defaultLogger) Info(msg string, args ...any) {
	Info(msg, args...)
}

func (d *defaultLogger) Debug(msg string, args ...any) {
	Debug(msg, args...)
}

func (d *defaultLogger) Warn(msg string, args ...any) {
	Warn(msg, args...)
}

func (d *defaultLogger) Error(msg string, args ...any) {
	Error(msg, args...)
}

func (d *defaultLogger) InfoPersist(msg string, args ...any) {
	InfoPersist(msg, args...)
}

func (d *defaultLogger) DebugPersist(msg string, args ...any) {
	DebugPersist(msg, args...)
}

func (d *defaultLogger) WarnPersist(msg string, args ...any) {
	WarnPersist(msg, args...)
}

func (d *defaultLogger) ErrorPersist(msg string, args ...any) {
	ErrorPersist(msg, args...)
}

func (d *defaultLogger) RecoverPanic(name string, cleanup func()) {
	RecoverPanic(name, cleanup)
}

func (d *defaultLogger) AppendToSessionLogFile(sessionId string, filename string, content string) string {
	return AppendToSessionLogFile(sessionId, filename, content)
}

func (d *defaultLogger) WriteRequestMessageJson(sessionId string, requestSeqId int, message any) string {
	return WriteRequestMessageJson(sessionId, requestSeqId, message)
}

func (d *defaultLogger) WriteRequestMessage(sessionId string, requestSeqId int, message string) string {
	return WriteRequestMessage(sessionId, requestSeqId, message)
}

func (d *defaultLogger) AppendToStreamSessionLogJson(sessionId string, requestSeqId int, jsonableChunk any) string {
	return AppendToStreamSessionLogJson(sessionId, requestSeqId, jsonableChunk)
}

func (d *defaultLogger) AppendToStreamSessionLog(sessionId string, requestSeqId int, chunk string) string {
	return AppendToStreamSessionLog(sessionId, requestSeqId, chunk)
}

func (d *defaultLogger) WriteChatResponseJson(sessionId string, requestSeqId int, response any) string {
	return WriteChatResponseJson(sessionId, requestSeqId, response)
}

func (d *defaultLogger) WriteToolResultsJson(sessionId string, requestSeqId int, toolResults any) string {
	return WriteToolResultsJson(sessionId, requestSeqId, toolResults)
}

// defaultLoggerInstance is the global default logger instance.
var defaultLoggerInstance Logger = &defaultLogger{}

// SetDefaultLogger sets the global default logger instance.
// This can be overridden in tests.
func SetDefaultLogger(l Logger) {
	defaultLoggerInstance = l
}

// GetDefaultLogger returns the global default logger instance.
func GetDefaultLogger() Logger {
	return defaultLoggerInstance
}
