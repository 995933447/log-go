package logger

import (
	"fmt"
	"log"
	"sync"
)

type Formatter interface {
	DisableCacheCaller(disabled bool)
	SetSkipCall(skipCall int)
	Copy() Formatter
	GetSkipCall() int
	Sprintf(level Level, stdoutColor Color, args ...interface{}) ([]byte, error)
}

type Msg struct {
	Level     Level
	SkipCall  int
	Formatted []byte
}

type Writer interface {
	IsLoggable(level Level) bool
	DisableCacheCaller(disabled bool)
	EnableStdoutPrinter()
	DisableStdoutPrinter()
	Write(level Level, args ...interface{}) error
	WriteBySkipCall(level Level, skipCall int, args ...interface{}) error
	WriteMsg(msg *Msg) error
	GetMsg(level Level, args ...interface{}) (*Msg, error)
	GetMsgBySkipCall(level Level, skipCall int, args ...interface{}) (*Msg, error)
	GetSkipCall() int
	Flush() error
}

type Logger struct {
	writer Writer
	mu     sync.RWMutex
}

func NewLogger(writer Writer) *Logger {
	return &Logger{
		writer: writer,
	}
}

func (l *Logger) EnableStdoutPrinter() {
	l.writer.EnableStdoutPrinter()
}

func (l *Logger) DisableStdoutPrinter() {
	l.writer.DisableStdoutPrinter()
}

func (l *Logger) GetWriter() Writer {
	return l.writer
}

func (l *Logger) Debug(content interface{}) {
	if err := l.Write(LevelDebug, content); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Info(content interface{}) {
	if err := l.Write(LevelInfo, content); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Important(content interface{}) {
	if err := l.Write(LevelImportant, content); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Warn(content interface{}) {
	if err := l.Write(LevelWarn, content); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Error(content interface{}) {
	if err := l.Write(LevelError, content); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Panic(content interface{}) {
	if err := l.Write(LevelPanic, content); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Fatal(content interface{}) {
	if err := l.Write(LevelFatal, content); err != nil {
		fmt.Println(err)
	}
	if err := l.Flush(); err != nil {
		fmt.Println(err)
	}
	log.Fatal(content)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	if err := l.Write(LevelDebug, append([]interface{}{format}, args...)...); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Infof(format string, args ...interface{}) {
	if err := l.Write(LevelInfo, append([]interface{}{format}, args...)...); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Importantf(format string, args ...interface{}) {
	if err := l.Write(LevelImportant, append([]interface{}{format}, args...)...); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	if err := l.Write(LevelWarn, append([]interface{}{format}, args...)...); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	if err := l.Write(LevelError, append([]interface{}{format}, args...)...); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Panicf(format string, args ...interface{}) {
	if err := l.Write(LevelPanic, append([]interface{}{format}, args...)...); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	if err := l.Write(LevelFatal, append([]interface{}{format}, args...)...); err != nil {
		fmt.Println(err)
	}
	if err := l.Flush(); err != nil {
		fmt.Println(err)
	}
	log.Fatalf(format, args...)
}

func (l *Logger) WriteBySkipCall(level Level, skipCall int, args ...interface{}) error {
	if err := l.writer.WriteBySkipCall(level, skipCall, args...); err != nil {
		return err
	}

	return nil
}

func (l *Logger) Write(level Level, args ...interface{}) error {
	if err := l.writer.Write(level, args...); err != nil {
		return err
	}

	return nil
}

func (l *Logger) Flush() error {
	return l.writer.Flush()
}
