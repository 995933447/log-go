package loggerwriter

import (
	"context"
	"fmt"
	"github.com/995933447/log-go"
	"github.com/995933447/log-go/impl/fmts"
	"github.com/995933447/std-go/print"
)

type StdoutLoggerWriter struct {
	fmt log.Formatter
	printColor print.Color
}

func NewStdoutLoggerWriter(printColor print.Color) *StdoutLoggerWriter {
	return &StdoutLoggerWriter{
		fmt:        fmts.NewSimpleTraceFormatter(4, fmts.FormatText),
		printColor: printColor,
	}
}

func (w *StdoutLoggerWriter) Write(ctx context.Context, level log.Level, format string, args ...interface{}) error {
	logContent, err := w.fmt.Sprintf(ctx, level, w.printColor, format, args...)
	if err != nil {
		return err
	}
	fmt.Println(logContent)
	return nil
}

func (*StdoutLoggerWriter) Flush() error {
	return nil
}
