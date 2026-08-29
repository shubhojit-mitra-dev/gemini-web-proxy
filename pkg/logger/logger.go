package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents log severity levels.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// String returns the string representation of the log level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Logger provides thread-safe structured logging capability.
type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	level  Level
	enable bool
}

var defaultLogger = New(os.Stderr, LevelInfo, true)

// New creates a new Logger instance.
func New(out io.Writer, level Level, enable bool) *Logger {
	return &Logger{
		out:    out,
		level:  level,
		enable: enable,
	}
}

// SetEnabled enables or disables logging output globally.
func SetEnabled(enabled bool) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.enable = enabled
}

// SetLevel sets the minimum log level for output.
func SetLevel(level Level) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.level = level
}

func (l *Logger) log(level Level, format string, v ...interface{}) {
	if !l.enable || level < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, v...)
	fmt.Fprintf(l.out, "[%s] [%s] %s\n", timestamp, level.String(), msg)
}

// Debug logs a message at LevelDebug.
func Debug(format string, v ...interface{}) {
	defaultLogger.log(LevelDebug, format, v...)
}

// Info logs a message at LevelInfo.
func Info(format string, v ...interface{}) {
	defaultLogger.log(LevelInfo, format, v...)
}

// Warn logs a message at LevelWarn.
func Warn(format string, v ...interface{}) {
	defaultLogger.log(LevelWarn, format, v...)
}

// Error logs a message at LevelError.
func Error(format string, v ...interface{}) {
	defaultLogger.log(LevelError, format, v...)
}
