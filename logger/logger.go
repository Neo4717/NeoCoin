package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type Level int8

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

var (
	levelNames = map[Level]string{
		LevelDebug: "DEBUG",
		LevelInfo:  "INFO",
		LevelWarn:  "WARN",
		LevelError: "ERROR",
		LevelFatal: "FATAL",
	}
)

type Fields map[string]any

type Logger struct {
	output   io.Writer
	minLevel Level
	mu       sync.Mutex
	caller   bool
	encoding string
}

type entry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Caller    string         `json:"caller,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
}

var defaultLogger *Logger
var once sync.Once

func Default() *Logger {
	once.Do(func() {
		defaultLogger = New(os.Stdout, LevelInfo)
	})
	return defaultLogger
}

func New(output io.Writer, minLevel Level) *Logger {
	return &Logger{
		output:   output,
		minLevel: minLevel,
		encoding: "json",
		caller:   true,
	}
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
}

func (l *Logger) SetCaller(enable bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.caller = enable
}

func (l *Logger) log(ctx context.Context, level Level, msg string, fields Fields) {
	if level < l.minLevel {
		return
	}

	e := entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     levelNames[level],
		Message:   msg,
	}

	if l.caller {
		if _, file, line, ok := runtime.Caller(2); ok {
			e.Caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
		}
	}

	if len(fields) > 0 {
		e.Fields = fields
	}

	if ctx != nil {
		if traceID := ctx.Value("trace_id"); traceID != nil {
			if e.Fields == nil {
				e.Fields = make(map[string]any)
			}
			e.Fields["trace_id"] = traceID
		}
	}

	data, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: marshal error: %v\n", err)
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.output.Write(data)
	l.output.Write([]byte{'\n'})
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...Fields) {
	l.log(ctx, LevelDebug, msg, mergeFields(fields...))
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...Fields) {
	l.log(ctx, LevelInfo, msg, mergeFields(fields...))
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...Fields) {
	l.log(ctx, LevelWarn, msg, mergeFields(fields...))
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...Fields) {
	l.log(ctx, LevelError, msg, mergeFields(fields...))
}

func (l *Logger) Fatal(ctx context.Context, msg string, fields ...Fields) {
	l.log(ctx, LevelFatal, msg, mergeFields(fields...))
	os.Exit(1)
}

func mergeFields(fs ...Fields) Fields {
	if len(fs) == 0 {
		return nil
	}
	if len(fs) == 1 {
		return fs[0]
	}

	merged := make(Fields)
	for _, f := range fs {
		for k, v := range f {
			merged[k] = v
		}
	}
	return merged
}

func WithField(key string, value any) Fields {
	return Fields{key: value}
}

func WithFields(fields map[string]any) Fields {
	return fields
}

func WithTraceID(traceID string) Fields {
	return Fields{"trace_id": traceID}
}

func WithError(err error) Fields {
	if err == nil {
		return nil
	}
	return Fields{"error": err.Error()}
}

func WithUserID(userID string) Fields {
	return Fields{"user_id": userID}
}

func Debug(msg string, fields ...Fields) { Default().Debug(context.Background(), msg, fields...) }
func Info(msg string, fields ...Fields)  { Default().Info(context.Background(), msg, fields...) }
func Warn(msg string, fields ...Fields)  { Default().Warn(context.Background(), msg, fields...) }
func Error(msg string, fields ...Fields) { Default().Error(context.Background(), msg, fields...) }
func Fatal(msg string, fields ...Fields) { Default().Fatal(context.Background(), msg, fields...) }
