package fmts

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/995933447/log-go/v2/loggo/logger"
	"github.com/995933447/runtimeutil"
)

type caller struct {
	fileName string
	funcName string
	line     int
}

var callerCache sync.Map

var _ logger.Formatter = (*TraceFormatter)(nil)

var replacer = strings.NewReplacer(
	"\n", `\n`,
	"\r", `\r`,
	"\t", `\t`,
)

func NewTraceFormatter(moduleName string, skipCall int, formatType Format, disabledStdoutColor, disabledCacheCaller bool, cfgLoader *logger.ConfLoader) *TraceFormatter {
	return &TraceFormatter{
		skipCall:            skipCall,
		formatType:          formatType,
		moduleName:          moduleName,
		cfgLoader:           cfgLoader,
		disabledStdoutColor: disabledStdoutColor,
		disabledCacheCaller: disabledCacheCaller,
	}
}

type TraceFormatter struct {
	skipCall            int
	formatType          Format
	moduleName          string
	cfgLoader           *logger.ConfLoader
	disabledStdoutColor bool
	disabledCacheCaller bool
}

func (f *TraceFormatter) DisableCacheCaller(disabled bool) {
	f.disabledCacheCaller = disabled
}

func (f *TraceFormatter) GetSkipCall() int {
	return f.skipCall
}

func (f *TraceFormatter) SetSkipCall(skipCall int) {
	f.skipCall = skipCall
}

func (f *TraceFormatter) Copy() logger.Formatter {
	return NewTraceFormatter(f.moduleName, f.skipCall, f.formatType, f.disabledStdoutColor, f.disabledCacheCaller, f.cfgLoader)
}

func (f *TraceFormatter) Sprintf(level logger.Level, stdoutColor logger.Color, args ...interface{}) ([]byte, error) {
	levelStr, err := logger.TransferLevelToStr(level)
	if err != nil {
		return nil, err
	}

	var colorStdoutStart, colorStdoutEnd string
	if !f.disabledStdoutColor && stdoutColor != logger.ColorNil {
		colorStdoutStart, err = logger.GetColorStdout(stdoutColor)
		if err != nil {
			return nil, err
		}
		colorStdoutEnd = logger.ColorToStdoutMap[logger.ColorNil]
	}

	var fileName, callFuncName string
	var callLine int
	if f.disabledCacheCaller {
		var (
			pc uintptr
			ok bool
		)
		pc, fileName, callLine, ok = runtime.Caller(f.skipCall)
		if ok {
			callFuncName = runtime.FuncForPC(pc).Name()
		}

		lastSlash := strings.LastIndexByte(fileName, '/')
		if lastSlash >= 0 {
			fileName = fileName[lastSlash+1:]
		}
	} else {
		rpc := make([]uintptr, 1)
		n := runtime.Callers(f.skipCall+1, rpc)
		if n > 0 {
			pc := rpc[0]
			callAny, ok := callerCache.Load(pc)
			if ok {
				call := callAny.(*caller)
				fileName, callLine, callFuncName = call.fileName, call.line, call.funcName
			} else {
				frame, _ := runtime.CallersFrames(rpc).Next()
				fileName = frame.File
				callFuncName = frame.Function
				callLine = frame.Line
				lastSlash := strings.LastIndexByte(fileName, '/')
				if lastSlash >= 0 {
					fileName = fileName[lastSlash+1:]
				}
				callerCache.Store(pc, &caller{
					fileName: fileName,
					line:     callLine,
					funcName: callFuncName,
				})
			}
		}
	}

	var rawFormatted string
	argsNum := len(args)
	if argsNum > 0 {
		firstArg := args[0]
		switch v := firstArg.(type) {
		case string:
			rawFormatted = v
		case fmt.Stringer:
			rawFormatted = v.String()
		case error:
			rawFormatted = v.Error()
		default:
			rawFormatted = fmt.Sprint(firstArg)
		}
		if argsNum > 1 {
			rawFormatted = fmt.Sprintf(rawFormatted, args[1:]...)
		}
		rawFormatted = replacer.Replace(rawFormatted)
	}

	trace, gid := runtimeutil.GetTraceWithGidDefNoTrace()

	switch level {
	case logger.LevelDebug:
		debugMsgMaxLen := f.cfgLoader.GetConf().File.DebugMsgMaxLen
		if debugMsgMaxLen > 0 {
			rawFormatted = f.truncateByRunes(rawFormatted, debugMsgMaxLen)
		}
	case logger.LevelInfo:
		infoMsgMaxLen := f.cfgLoader.GetConf().File.InfoMsgMaxLen
		if infoMsgMaxLen > 0 {
			rawFormatted = f.truncateByRunes(rawFormatted, infoMsgMaxLen)
		}
	}

	callLineStr := strconv.Itoa(callLine)
	// App id & Uid & Gid
	var buf [19]byte
	bb := buf[:0]
	bb = strconv.AppendInt(bb, gid, 10)

	if f.formatType == FormatText {
		// 预估分配
		msgLen := 40 + len(rawFormatted) + len(f.moduleName) + len(trace) +
			len(colorStdoutStart) + len(colorStdoutEnd) + len(levelStr) + len(fileName) +
			len(callFuncName) + len(callLineStr) + len(bb)
		b := make([]byte, 0, msgLen)

		// Timestamp
		b = append(b, '[')
		b = append(b, runtimeutil.GetNowFormatFast()...)
		nano := runtimeutil.GetNowNanosecondFast() / 100000
		if nano < 10 {
			b = append(b, '.', '0', '0', '0')
		} else if nano < 100 {
			b = append(b, '.', '0', '0')
		} else if nano < 1000 {
			b = append(b, '.', '0')
		} else {
			b = append(b, '.')
		}
		b = strconv.AppendInt(b, int64(nano), 10)
		b = append(b, "] ["...)

		// Module name
		b = append(b, f.moduleName...)
		b = append(b, "] ["...)

		// Trace info
		b = append(b, trace...)
		b = append(b, "]["...)

		// Gid
		b = append(b, bb...)

		b = append(b, "] "...)

		// Color & level
		b = append(b, colorStdoutStart...)
		b = append(b, levelStr...)
		b = append(b, ' ')

		// Caller info
		b = append(b, callFuncName...)
		b = append(b, ':')
		b = append(b, fileName...)
		b = append(b, ':')
		b = append(b, callLineStr...)
		b = append(b, colorStdoutEnd...)

		// Log message
		b = append(b, ' ')
		b = append(b, rawFormatted...)
		b = append(b, '\n')

		return b, nil
	}

	return nil, errors.New("not support log format")
}

func (f *TraceFormatter) truncateByRunes(s string, maxLen int32) string {
	if maxLen <= 0 {
		return s
	}

	var i, runeCount int32
	sLen := int32(len(s))
	for i < sLen && runeCount < maxLen {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += int32(size)
		runeCount++
	}

	if runeCount == maxLen && i < sLen {
		return s[:i] + "..."
	}
	return s
}
