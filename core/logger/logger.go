package logger

import (
	"io"
	"os"
)

const (
	/* Default errors */
	ERR_UnableToLogContent = "Unable to log content."

	/* Default logging levels */
	LOG_Information 	= 0
	LOG_Warning 		= 1
	LOG_Error 			= 2
	LOG_Critical		= 3
	LOG_Fatal			= 4
)

var (
	LoggingHeaders = map[int][]byte {
		LOG_Information: []byte("Information: "),
		LOG_Warning: []byte("Warning: "),
		LOG_Error: []byte("Error: "),
		LOG_Critical: []byte("Critical"),
		LOG_Fatal: []byte("Fatal: "),
	}
)

/* Logger type definition */
type Logger struct {
	writer	io.Writer
	minLoggingLevel int
}

/* Instantiates a new logger to NULL */
func NewNullLogger() *Logger {
	return &Logger{nil, LOG_Information}
}

/* Instantiate a new simple logger (logs to stdout) */
func NewSimpleLogger() *Logger {
	return NewLogger(os.Stdout)
}

/* Instantiates a new logger supporting multiple write sources */
func NewLogger(writers ...io.Writer) *Logger {
	return &Logger{io.MultiWriter(writers...), LOG_Information}
}

/* Logger's writer function */
func (l *Logger) _write(message []byte) (bool, error) {
	n, err := l.writer.Write(append(message, '\n'))
	return n > 0, err
}

/* Logger's writer function wrapper */
func (l *Logger) write(level int, message string) {
	if l.writer != nil && l.minLoggingLevel <= level && len(message) > 0 {
		ok, _ := l._write(append(LoggingHeaders[level], []byte(message)...))

		if !ok {
			l._write(append(LoggingHeaders[LOG_Critical], ERR_UnableToLogContent...))
		} else {
			if level == LOG_Fatal {
				os.Exit(1)
			}
		}
	}
}

/* Log a information */
func (l *Logger) Information(message string) {
	l.write(LOG_Information, message)
}

/* Log a information */
func (l *Logger) Warning(message string) {
	l.write(LOG_Warning, message)
}

/* Log a information */
func (l *Logger) Error(message string) {
	l.write(LOG_Error, message)
}

/* Log a information */
func (l *Logger) Critical(message string) {
	l.write(LOG_Critical, message)
}

/* Log a information */
func (l *Logger) Fatal(message string) {
	l.write(LOG_Fatal, message)
}

/* Limit the written logs based on importance level. */
func (l *Logger) LimitLoggingLevel(level int) {
	if level >= LOG_Information && level <= LOG_Fatal {
		l.minLoggingLevel = level
	}
}
