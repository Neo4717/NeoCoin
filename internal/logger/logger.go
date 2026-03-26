package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
	PANIC
)

func (l Level) String() string {
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
	case PANIC:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

func ParseLevel(s string) Level {
	switch s {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	case "panic":
		return PANIC
	default:
		return INFO
	}
}

type OutputEncoding int

const (
	TextEncoding OutputEncoding = iota
	JSONEncoding
)

type Logger struct {
	level            Level
	encoding         OutputEncoding
	outputFile       string
	fileHandle       *os.File
	timestampFormat  string
	enableCallerInfo bool

	mu     sync.Mutex
	writer *sync.Map
}

var log *Logger
var once sync.Once

func NewLogger(level string, encoding OutputEncoding, outputFile string, timestampFormat string, enableCallerInfo bool) *Logger {
	l := &Logger{
		level:            ParseLevel(level),
		encoding:         encoding,
		outputFile:       outputFile,
		timestampFormat:  timestampFormat,
		enableCallerInfo: enableCallerInfo,
	}

	if timestampFormat == "" {
		l.timestampFormat = time.RFC3339
	}

	if outputFile != "" {
		l.initFileOutput()
	}

	return l
}

func (l *Logger) initFileOutput() {
	once.Do(func() {
		dir := filepath.Dir(l.outputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "logger: failed to create log directory: %v\n", err)
			return
		}

		f, err := os.OpenFile(l.outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "logger: failed to open log file: %v\n", err)
			return
		}
		l.fileHandle = f
		go l.rotateFile()
	})
}

func (l *Logger) rotateFile() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		if l.fileHandle != nil {
			f, err := os.OpenFile(l.outputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err == nil {
				l.fileHandle.Close()
				l.fileHandle = f
			}
		}
		l.mu.Unlock()
	}
}

func Init(level string, encoding OutputEncoding, outputFile string, timestampFormat string, enableCallerInfo bool) {
	log = NewLogger(level, encoding, outputFile, timestampFormat, enableCallerInfo)
}

func InitWithConfig(cfg map[string]interface{}) {
	level := "info"
	encoding := TextEncoding
	outputFile := ""
	timestampFormat := ""
	enableCallerInfo := false

	if v, ok := cfg["level"].(string); ok {
		level = v
	}
	if v, ok := cfg["encoding"].(string); ok && v == "json" {
		encoding = JSONEncoding
	}
	if v, ok := cfg["outputFile"].(string); ok {
		outputFile = v
	}
	if v, ok := cfg["timestampFormat"].(string); ok {
		timestampFormat = v
	}
	if v, ok := cfg["enableCallerInfo"].(bool); ok {
		enableCallerInfo = v
	}

	log = NewLogger(level, encoding, outputFile, timestampFormat, enableCallerInfo)
}

func (l *Logger) log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	callerInfo := ""
	if l.enableCallerInfo {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			callerInfo = fmt.Sprintf(" [%s:%d]", filepath.Base(file), line)
		}
	}

	timestamp := time.Now().Format(l.timestampFormat)
	entry := fmt.Sprintf("%s [%s]%s %s", timestamp, level.String(), callerInfo, msg)

	if l.encoding == JSONEncoding {
		jsonEntry := l.formatJSON(timestamp, level.String(), msg, callerInfo)
		l.writeOutput(jsonEntry)
		return
	}
	l.writeOutput(entry)
}

func (l *Logger) formatJSON(timestamp, level, msg, caller string) string {
	if caller != "" {
		return fmt.Sprintf(`{"time":"%s","level":"%s","caller":"%s","msg":"%s"}`,
			timestamp, level, strings.TrimSpace(caller), msg)
	}
	return fmt.Sprintf(`{"time":"%s","level":"%s","msg":"%s"}`,
		timestamp, level, msg)
}

func (l *Logger) writeOutput(entry string) {
	fmt.Fprintln(os.Stderr, entry)
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.fileHandle != nil {
		fmt.Fprintln(l.fileHandle, entry)
	}
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(DEBUG, msg, args...)
}

func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(INFO, msg, args...)
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(WARN, msg, args...)
}

func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(ERROR, msg, args...)
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(FATAL, msg, args...)
	os.Exit(1)
}

func (l *Logger) Panic(msg string, args ...interface{}) {
	l.log(PANIC, msg, args...)
	panic(fmt.Sprintf(msg, args...))
}

func Debug(msg string, args ...interface{}) {
	log.Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	log.Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	log.Warn(msg, args...)
}

func Error(msg string, args ...interface{}) {
	log.Error(msg, args...)
}

func Fatal(msg string, args ...interface{}) {
	log.Fatal(msg, args...)
}

func Panic(msg string, args ...interface{}) {
	log.Panic(msg, args...)
}
