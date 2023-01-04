package log

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type LoggerWriter interface {
	Write(ctx context.Context, level Level, format string, args ...interface{}) error
	Flush() error
}

type Logger struct {
	logLevel Level
	loggerWriter LoggerWriter
	mu sync.RWMutex
}

func NewLogger(loggerWriter LoggerWriter) *Logger {
	return &Logger{
		loggerWriter: loggerWriter,
	}
}

func (l *Logger) SetLogLevel(level Level)  {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logLevel = level
}

func (l *Logger) GetLogLevel() Level {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.logLevel
}

func (l *Logger) Debug(ctx context.Context, content interface{}) {
	if err := l.write(ctx, LevelDebug, content); err != nil {
		panic(err)
	}
}

func (l *Logger) Info(ctx context.Context, content interface{}) {
	if err := l.write(ctx, LevelInfo, content); err != nil {
		panic(err)
	}
}

func (l *Logger) Warn(ctx context.Context, content interface{}) {
	if err := l.write(ctx, LevelDebug, content); err != nil {
		panic(err)
	}
}

func (l *Logger) Error(ctx context.Context, content interface{}) {
	if err := l.write(ctx, LevelError, content); err != nil {
		panic(err)
	}
}

func (l *Logger) Fatal(ctx context.Context, content interface{}) {
	if err := l.write(ctx, LevelFatal, content); err != nil {
		panic(err)
	}
}

func (l *Logger) Panic(ctx context.Context, content interface{}) {
	if err := l.write(ctx, LevelPanic, content); err != nil {
		panic(err)
	}
}

func (l *Logger) Debugf(ctx context.Context, format string, args ...interface{}) {
	if err := l.write(ctx, LevelDebug, append([]interface{}{format}, args...)...); err != nil {
		panic(err)
	}
}

func (l *Logger) Infof(ctx context.Context, format string, args ...interface{}) {
	if err := l.write(ctx, LevelInfo, append([]interface{}{format}, args...)...); err != nil {
		panic(err)
	}
}

func (l *Logger) Warnf(ctx context.Context, format string, args ...interface{}) {
	if err := l.write(ctx, LevelWarn, append([]interface{}{format}, args...)...); err != nil {
		panic(err)
	}
}

func (l *Logger) Errorf(ctx context.Context, format string, args ...interface{}) {
	if err := l.write(ctx, LevelError, append([]interface{}{format}, args...)...); err != nil {
		panic(err)
	}
}

func (l *Logger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	if err := l.write(ctx, LevelFatal, append([]interface{}{format}, args...)...); err != nil {
		panic(err)
	}
}

func (l *Logger) Panicf(ctx context.Context, format string, args ...interface{}) {
	if err := l.write(ctx, LevelPanic, append([]interface{}{format}, args...)...); err != nil {
		panic(err)
	}
}

func (l *Logger) write(ctx context.Context, level Level, args ...interface{}) error {
	if l.logLevel > level {
		return nil
	}

	argNum := len(args)
	if argNum == 0 {
		return errors.New("args num is 0")
	}

	var realArgs []interface{}
	if argNum > 1 {
		realArgs = args[1:]
	}

	format := fmt.Sprintf("%s", args[0])

	if err := l.loggerWriter.Write(ctx, level, format, realArgs...); err != nil {
		return err
	}

	return nil
}

func (l *Logger) Flush() error {
	return l.loggerWriter.Flush()
}