package fmts

import (
	"context"
	"errors"
	"fmt"
	"github.com/995933447/log-go"
	simpletracectx "github.com/995933447/simpletrace/context"
	"github.com/995933447/std-go/print"
	"runtime"
	"time"
)

type SimpleTraceFormatter struct {
	skipCall int
	formatType Format
}

var _ log.Formatter = (*SimpleTraceFormatter)(nil)

func NewSimpleTraceFormatter(skipCall int, formatType Format) *SimpleTraceFormatter {
	return &SimpleTraceFormatter{
		skipCall: skipCall,
		formatType: formatType,
	}
}

func (f *SimpleTraceFormatter) Sprintf(ctx context.Context, level log.Level, stdoutColor print.Color, format string, args ...interface{}) (string, error) {
	levelStr, err := log.TransferLevelToStr(level)
	if err != nil {
		return "", err
	}

	colorStdout, err := print.GetColorStdout(stdoutColor)
	if err != nil {
		return "", err
	}

	pc, callFile, callLine, ok := runtime.Caller(f.skipCall)
	var callFuncName string
	if ok {
		callFuncName = runtime.FuncForPC(pc).Name()
	}

	var (
		moduleName string
		traceId, spanId, parentSpanId string
	)
	if traceCtx, ok := ctx.(*simpletracectx.Context); ok {
		parentSpanId = traceCtx.GetParentSpanId()
		spanId = traceCtx.GetSpanId()
		traceId = traceCtx.GetTraceId()
		moduleName = traceCtx.GetModuleName()
		if traceCtx.GetParentSpanId() == "" {
			parentSpanId = "none"
		}
	} else {
		spanId = "none"
		parentSpanId = "none"
		traceId = "none"
		moduleName = "(unknown)"
	}

	now := time.Now()
	rawFormatted := fmt.Sprintf(format, args...)

	switch f.formatType {
	case FormatText:
		return fmt.Sprintf(
			"[%s] %s.%04d <t:%s,s:%s,ps:%s> %s%s %s:%d:%s \u001B[0m%s\n",
			moduleName,
			now.Format("01-02T15:04:05"),
			now.Nanosecond() / 100000,
			traceId,
			spanId,
			parentSpanId,
			colorStdout,
			levelStr,
			callFile,
			callLine,
			callFuncName,
			rawFormatted,
		), nil
	default:
		return "", errors.New("not support log format")
	}
}
