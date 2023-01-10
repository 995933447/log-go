package loggerwriter

import (
	"context"
	"errors"
	"fmt"
	"github.com/995933447/log-go"
	"github.com/995933447/log-go/impl/fmts"
	"github.com/995933447/std-go/print"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var levelToStdoutColorMap = map[log.Level]print.Color{
	log.LevelDebug: print.ColorBlue,
	log.LevelInfo: print.ColorGreen,
	log.LevelWarn: print.ColorYellow,
	log.LevelError: print.ColorRed,
	log.LevelFatal: print.ColorPurple,
	log.LevelPanic: print.ColorRed,
}

type CheckTimeToOpenNewFileFunc func(lastOpenFileTime *time.Time, isNeverOpenFile bool) (string, bool)

var OpenNewFileByByDateHour CheckTimeToOpenNewFileFunc  = func (lastOpenFileTime *time.Time, isNeverOpenFile bool) (string, bool) {
	if isNeverOpenFile {
		return time.Now().Format("2006010215.log"), true
	}

	if lastOpenFileTime.Hour() != time.Now().Hour() {
		return time.Now().Format("2006010215.log"), true
	}

	lastOpenYear, lastOpenMonth, lastOpenDay := lastOpenFileTime.Date()
	nowYear, nowMonth, nowDay := time.Now().Date()
	if lastOpenDay != nowDay || lastOpenMonth != nowMonth || lastOpenYear != nowYear {
		return time.Now().Format("2006010215.log"), true
	}

	return "", false
}

type FileLoggerWriter struct {
	enabledStdoutPrinter atomic.Bool
	fp *os.File
	baseDir string
	maxFileSize int64
	lastCheckIsFullAt int64
	isFileFull bool
	checkFileFullIntervalSecs int64
	checkTimeToOpenNewFile CheckTimeToOpenNewFileFunc
	openCurrentFileTime *time.Time
	currentFileName string
	fmt log.Formatter
	bufCh chan []byte
	isFlushing atomic.Value
	flushSignCh chan struct{}
	flushDoneSignCh chan error
}

var _ log.LoggerWriter = (*FileLoggerWriter)(nil)

func NewFileLoggerWriter(baseDir string, maxFileSize int64, checkFileFullIntervalSecs int64, checkTimeToOpenNewFile CheckTimeToOpenNewFileFunc, bufChanLen uint32) *FileLoggerWriter {
	return &FileLoggerWriter{
		baseDir: strings.TrimRight(baseDir, "/"),
		maxFileSize: maxFileSize,
		checkFileFullIntervalSecs: checkFileFullIntervalSecs,
		checkTimeToOpenNewFile: checkTimeToOpenNewFile,
		fmt: fmts.NewSimpleTraceFormatter(4, fmts.FormatText),
		bufCh: make(chan []byte, bufChanLen),
		flushSignCh: make(chan struct{}),
		flushDoneSignCh: make(chan error),
	}
}

func (w *FileLoggerWriter) EnableStdoutPrinter() {
	w.enabledStdoutPrinter.Store(true)
}

func (w *FileLoggerWriter) DisableStdoutPrinter() {
	w.enabledStdoutPrinter.Store(false)
}

func (w *FileLoggerWriter) SetFormatter(fmt log.Formatter) *FileLoggerWriter {
	w.fmt = fmt
	return w
}

func (w *FileLoggerWriter) checkFileIsFull() (bool, error) {
	if w.lastCheckIsFullAt + w.checkFileFullIntervalSecs < time.Now().Unix() {
		return w.isFileFull, nil
	}

	fileInfo, err := w.fp.Stat()
	if err != nil {
		return false, err
	}

	w.isFileFull = fileInfo.Size() >= w.maxFileSize
	w.lastCheckIsFullAt = time.Now().Unix()

	return w.isFileFull, nil
}

func (w *FileLoggerWriter) tryOpenNewFile() error {
	var err error
	fileName, ok := w.checkTimeToOpenNewFile(w.openCurrentFileTime, w.openCurrentFileTime == nil)
	if !ok {
		if w.fp == nil {
			return errors.New("get first file name failed")
		}

		return nil
	}

	if w.fp == nil {
		if _, err = os.Stat(w.baseDir); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			if err = os.MkdirAll(w.baseDir, 0755); err != nil {
				return err
			}
		}
	}

	if w.fp, err = os.OpenFile(w.baseDir + "/" + fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755); err != nil{
		return err
	}

	openFileTime := time.Now()
	w.openCurrentFileTime = &openFileTime
	w.isFileFull = false
	w.lastCheckIsFullAt = 0
	w.currentFileName = fileName

	return nil
}

func (w *FileLoggerWriter) Write(ctx context.Context, level log.Level, format string, args ...interface{}) error {
	stdoutColor, ok := levelToStdoutColorMap[level]
	if !ok {
		stdoutColor = print.ColorNil
	}

	logContent, err := w.fmt.Sprintf(ctx, level, stdoutColor, format, args...)
	if err != nil {
		return err
	}

	if w.enabledStdoutPrinter.Load() {
		fmt.Print(logContent)
	}

	w.bufCh <- []byte(logContent)

	return nil
}

func (w *FileLoggerWriter) Flush() error {
	w.isFlushing.Store(true)
	w.flushSignCh <- struct{}{}
	return <- w.flushDoneSignCh
}

func (w *FileLoggerWriter) finishFlush(err error) {
	w.isFlushing.Store(false)
	w.flushDoneSignCh <- err
}

func (w *FileLoggerWriter) isFlushingNow() bool {
	return w.isFlushing.Load().(bool)
}

func (w *FileLoggerWriter) Loop() error {
	doWriteMoreAsPossible := func(buf []byte) error {
		for {
			var moreBuf []byte
			select {
			case moreBuf = <- w.bufCh:
				buf = append(buf, moreBuf...)
			default:
			}

			if moreBuf == nil {
				break
			}
		}
		
		if len(buf) == 0 {
			return nil
		}

		if err := w.tryOpenNewFile(); err != nil {
			return err
		}

		if isFull, err := w.checkFileIsFull(); err != nil  {
			return err
		} else if isFull {
			fmt.Printf("log file %s is overflow max size %d bytes.", w.currentFileName, w.maxFileSize)
			return nil
		}

		bufLen := len(buf)
		var totalWrittenBytes int
		for {
			n, err := w.fp.Write(buf[totalWrittenBytes:])
			if err != nil {
				return err
			}
			totalWrittenBytes += n
			if totalWrittenBytes >= bufLen {
				break
			}
		}

		return nil
	}

	for {
		select {
			case buf := <- w.bufCh:
				if err := doWriteMoreAsPossible(buf); err != nil {
					return err
				}
			case _ = <-w.flushSignCh:
				if err := doWriteMoreAsPossible([]byte{}); err != nil {
					w.finishFlush(err)
					break
				}
				if err := w.fp.Sync(); err != nil {
					w.finishFlush(err)
					break
				}
				w.finishFlush(nil)
		}
	}
}