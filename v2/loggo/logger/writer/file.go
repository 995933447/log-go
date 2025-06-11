package writer

import (
	"errors"
	"fmt"
	"github.com/995933447/gofiler"
	logger2 "github.com/995933447/log-go/v2/loggo/logger"
	fmts2 "github.com/995933447/log-go/v2/loggo/logger/fmts"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

const (
	FileSuffix           = ".txt"
	CompressedFileSuffix = ".zip"
)

var levelToStdoutColorMap = map[logger2.Level]logger2.Color{
	logger2.LevelDebug:     logger2.ColorLightGreen,
	logger2.LevelInfo:      logger2.ColorLightGreen,
	logger2.LevelImportant: logger2.ColorBlue,
	logger2.LevelWarn:      logger2.ColorGreen,
	logger2.LevelError:     logger2.ColorRed,
	logger2.LevelPanic:     logger2.ColorRed,
	logger2.LevelFatal:     logger2.ColorPurple,
}

type CheckTimeToOpenNewFileFunc func(writer *FileWriter, lastOpenFileTime *time.Time, isNeverOpenFile bool) (string, bool)

var OpenNewFileByByDateHour CheckTimeToOpenNewFileFunc = func(writer *FileWriter, lastOpenFileTime *time.Time, isNeverOpenFile bool) (string, bool) {
	fileName := writer.getFilePrefix() + time.Now().Format("2006010215") + "_" + FileSuffix

	if writer.fp == nil {
		return fileName, true
	}

	if isNeverOpenFile {
		return fileName, true
	}

	if lastOpenFileTime.Hour() != time.Now().Hour() {
		return fileName, true
	}

	lastOpenYear, lastOpenMonth, lastOpenDay := lastOpenFileTime.Date()
	nowYear, nowMonth, nowDay := time.Now().Date()
	if lastOpenDay != nowDay || lastOpenMonth != nowMonth || lastOpenYear != nowYear {
		return fileName, true
	}

	return "", false
}

type FileLogConf struct {
	MaxFileSize int64
}

var _ logger2.Writer = (*FileWriter)(nil)

type FileWriterConf struct {
	ModuleName, BaseDir, FilePrefix string
	SkipCall                        int
	LogCfgLoader                    *logger2.ConfLoader
	CheckFileFullIntervalSec        int64
	CheckTimeToOpenNewFile          CheckTimeToOpenNewFileFunc
	BufChanLen                      uint32
	OnLogErr                        func(err error)
}

func (c *FileWriterConf) Check() error {
	if c.BaseDir == "" {
		c.BaseDir = "."
	}
	if c.SkipCall == 0 {
		return errors.New("SkipCall is zero")
	}
	if c.LogCfgLoader == nil {
		return errors.New("LogCfgLoader is nil")
	}
	if c.CheckTimeToOpenNewFile == nil {
		c.CheckTimeToOpenNewFile = OpenNewFileByByDateHour
	}
	return nil
}

func NewFileWriter(cfg *FileWriterConf) (*FileWriter, error) {
	cfg.BaseDir = strings.TrimRight(cfg.BaseDir, "/")
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return &FileWriter{
		cfg:             cfg,
		fmt:             fmts2.NewTraceFormatter(cfg.ModuleName, cfg.SkipCall, fmts2.FormatText, false, cfg.LogCfgLoader),
		bufCh:           make(chan []byte, cfg.BufChanLen),
		flushSignCh:     make(chan struct{}),
		flushDoneSignCh: make(chan error),
	}, nil
}

type FileWriter struct {
	cfg                  *FileWriterConf
	enabledStdoutPrinter atomic.Bool
	fp                   *os.File
	curSizeBytes         int64
	lastCheckIsFullAt    int64
	isFileFull           bool
	isWrittenFullTip     bool
	openCurFileTime      *time.Time
	curFileName          string
	fmt                  logger2.Formatter
	bufCh                chan []byte
	isFlushing           atomic.Bool
	flushSignCh          chan struct{}
	flushDoneSignCh      chan error
	isHandlingExpiredLog atomic.Bool
	lastFullBufChTipAt   atomic.Int64
}

func (w *FileWriter) GetCurFileName() string {
	return w.curFileName
}

func (w *FileWriter) GetFilePrefix() string {
	return w.getFilePrefix()
}

func (w *FileWriter) getFilePrefix() string {
	filePrefix := w.cfg.LogCfgLoader.GetConf().File.FilePrefix
	if filePrefix == "" {
		filePrefix = w.cfg.FilePrefix
	}
	if filePrefix != "" {
		filePrefix += "."
	}
	return filePrefix
}

func (w *FileWriter) GetSkipCall() int {
	return w.fmt.GetSkipCall()
}

func (w *FileWriter) EnableStdoutPrinter() {
	w.enabledStdoutPrinter.Store(true)
}

func (w *FileWriter) DisableStdoutPrinter() {
	w.enabledStdoutPrinter.Store(false)
}

func (w *FileWriter) SetFormatter(fmt logger2.Formatter) *FileWriter {
	w.fmt = fmt
	return w
}

func (w *FileWriter) checkFileIsFull() (bool, error) {
	if w.lastCheckIsFullAt != 0 && w.lastCheckIsFullAt+w.cfg.CheckFileFullIntervalSec > time.Now().Unix() {
		return w.isFileFull, nil
	}

	cfg := w.cfg.LogCfgLoader.GetConf()
	if cfg.File.MaxFileSizeBytes > 0 {
		fileName := w.fp.Name()
		fileInfo, err := os.Stat(w.fp.Name())
		if err != nil {
			if !os.IsNotExist(err) {
				return false, err
			}

			// 文件被删除了
			if w.fp, err = os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755); err != nil {
				return false, err
			}

			fileInfo, err = w.fp.Stat()
			if err != nil {
				return false, err
			}
		}

		w.curSizeBytes = fileInfo.Size()
		w.isFileFull = w.curSizeBytes >= cfg.File.MaxFileSizeBytes
		w.lastCheckIsFullAt = time.Now().Unix()
	}

	return w.isFileFull, nil
}

func (w *FileWriter) tryOpenNewFile() error {
	var err error
	fileName, ok := w.cfg.CheckTimeToOpenNewFile(w, w.openCurFileTime, w.openCurFileTime == nil)
	if !ok {
		if w.fp == nil {
			return errors.New("get first file name failed")
		}

		return nil
	}

	if w.fp == nil {
		_, err = os.Stat(w.cfg.BaseDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			if err = os.MkdirAll(w.cfg.BaseDir, 0755); err != nil {
				return err
			}
		}
	}

	if w.fp, err = os.OpenFile(w.cfg.BaseDir+"/"+fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755); err != nil {
		return err
	}

	fileInfo, err := w.fp.Stat()
	if err != nil {
		return err
	}

	w.curSizeBytes = fileInfo.Size()
	openFileTime := time.Now()
	w.openCurFileTime = &openFileTime
	w.isFileFull = false
	w.lastCheckIsFullAt = 0
	w.curFileName = fileName

	return nil
}

func (w *FileWriter) isLoggable(level logger2.Level) bool {
	if level < w.cfg.LogCfgLoader.GetConf().File.GetLevel() {
		return false
	}

	limitedBytes := int64(-1)
	switch level {
	case logger2.LevelDebug:
		limitedBytes = w.cfg.LogCfgLoader.GetConf().File.LogDebugBeforeFileSizeBytes
	case logger2.LevelInfo:
		limitedBytes = w.cfg.LogCfgLoader.GetConf().File.LogInfoBeforeFileSizeBytes
	}
	if limitedBytes >= 0 && w.curSizeBytes >= limitedBytes {
		return false
	}

	return true
}

func (w *FileWriter) asyncWrite(logContent string) {
	select {
	case w.bufCh <- []byte(logContent):
	default:
		if time.Now().Unix()-w.lastFullBufChTipAt.Load() > 5 {
			fmt.Println("log chan is full, content:", logContent)
			w.lastFullBufChTipAt.Store(time.Now().Unix())
		}
	}
}

func (w *FileWriter) WriteBySkipCall(level logger2.Level, skipCall int, format string, args ...interface{}) error {
	if !w.isLoggable(level) {
		return nil
	}

	stdoutColor, ok := levelToStdoutColorMap[level]
	if !ok {
		stdoutColor = logger2.ColorNil
	}

	fm := w.fmt
	if w.fmt.GetSkipCall() != skipCall {
		fm = w.fmt.Copy()
		fm.SetSkipCall(skipCall)
	}

	logContent, err := fm.Sprintf(level, stdoutColor, format, args...)
	if err != nil {
		return err
	}

	if w.enabledStdoutPrinter.Load() {
		fmt.Print(logContent)
	}

	w.asyncWrite(logContent)

	return nil
}

func (w *FileWriter) Write(level logger2.Level, format string, args ...interface{}) error {
	if !w.isLoggable(level) {
		return nil
	}

	stdoutColor, ok := levelToStdoutColorMap[level]
	if !ok {
		stdoutColor = logger2.ColorNil
	}

	logContent, err := w.fmt.Sprintf(level, stdoutColor, format, args...)
	if err != nil {
		return err
	}

	if w.enabledStdoutPrinter.Load() {
		fmt.Print(logContent)
	}

	w.asyncWrite(logContent)

	return nil
}

func (w *FileWriter) WriteMsg(msg *logger2.Msg) error {
	if !w.isLoggable(msg.Level) {
		return nil
	}

	w.asyncWrite(msg.Formatted)

	return nil
}

func (w *FileWriter) GetMsg(level logger2.Level, format string, args ...interface{}) (*logger2.Msg, error) {
	stdoutColor, ok := levelToStdoutColorMap[level]
	if !ok {
		stdoutColor = logger2.ColorNil
	}

	formatted, err := w.fmt.Sprintf(level, stdoutColor, format, args...)
	if err != nil {
		return nil, err
	}

	return &logger2.Msg{
		Level:     level,
		Format:    format,
		Args:      args,
		SkipCall:  w.fmt.GetSkipCall(),
		Formatted: formatted,
	}, nil
}

func (w *FileWriter) GetMsgBySkipCall(level logger2.Level, skipCall int, format string, args ...interface{}) (*logger2.Msg, error) {
	stdoutColor, ok := levelToStdoutColorMap[level]
	if !ok {
		stdoutColor = logger2.ColorNil
	}

	fm := w.fmt
	if w.fmt.GetSkipCall() != skipCall {
		fm = w.fmt.Copy()
		fm.SetSkipCall(skipCall)
	}

	formatted, err := fm.Sprintf(level, stdoutColor, format, args...)
	if err != nil {
		return nil, err
	}

	return &logger2.Msg{
		Level:     level,
		Format:    format,
		Args:      args,
		SkipCall:  skipCall,
		Formatted: formatted,
	}, nil
}

func (w *FileWriter) Flush() error {
	w.isFlushing.Store(true)
	w.flushSignCh <- struct{}{}
	return <-w.flushDoneSignCh
}

func (w *FileWriter) finishFlush(err error) {
	w.isFlushing.Store(false)
	w.flushDoneSignCh <- err
}

func (w *FileWriter) isFlushingNow() bool {
	return w.isFlushing.Load()
}

func (w *FileWriter) hdlExpiredFiles() {
	logCfg := w.cfg.LogCfgLoader.GetConf().File

	if w.isHandlingExpiredLog.Load() {
		return
	}

	w.isHandlingExpiredLog.Store(true)
	defer w.isHandlingExpiredLog.Store(false)

	_ = filepath.Walk(w.cfg.BaseDir, func(path string, info os.FileInfo, err error) error {
		//defer func() {
		//	if r := recover(); r != nil {
		//		fmt.Println(fmt.Errorf("unable to handle expired log '%s', error: %+v", path, r))
		//	}
		//}()

		if info == nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if logCfg.FileMaxRemainDays > 0 && info.ModTime().Unix() < (time.Now().Unix()-3600*24*int64(logCfg.FileMaxRemainDays)) {
			if strings.HasPrefix(filepath.Base(path), w.getFilePrefix()) && (strings.HasSuffix(path, FileSuffix) || strings.HasSuffix(path, CompressedFileSuffix)) {
				if err := os.Remove(path); err != nil {
					fmt.Println(err)
				}
				return nil
			}
		}

		if logCfg.CompressFrequentHours > 0 && info.ModTime().Unix() < (time.Now().Unix()-3600*int64(logCfg.CompressFrequentHours)) {
			if logCfg.CompressAfterReachBytes > 0 && info.Size() < logCfg.CompressAfterReachBytes {
				return nil
			}

			if strings.HasPrefix(filepath.Base(path), w.getFilePrefix()) && strings.HasSuffix(path, FileSuffix) {
				file, err := os.OpenFile(path, os.O_RDWR, os.ModePerm)
				if err != nil {
					fmt.Println(err)
					return nil
				}

				defer file.Close()

				err = gofiler.Zip([]*os.File{file}, file.Name()+".zip")
				if err != nil {
					fmt.Println(err)
					return nil
				}

				if err := os.Remove(path); err != nil {
					fmt.Println(err)
				}

				return nil
			}
		}

		return nil
	})
}

func (w *FileWriter) Loop() {
	doWriteMoreAsPossible := func(buf []byte) error {
		for {
			var moreBuf []byte
			select {
			case moreBuf = <-w.bufCh:
				buf = append(buf, moreBuf...)
			default:
			}

			if moreBuf == nil || len(buf) > 1024*16 {
				break
			}
		}

		if len(buf) == 0 {
			return nil
		}

		if err := w.tryOpenNewFile(); err != nil {
			return err
		}

		isFull, err := w.checkFileIsFull()
		if err != nil {
			return err
		}

		if isFull {
			if w.isWrittenFullTip {
				return nil
			}
			buf = []byte(fmt.Sprintf("%s文件已超出当前小时允许最大尺寸:%d bytes!!!\u001B[0m\n", logger2.ColorToStdoutMap[logger2.ColorPurple], w.cfg.LogCfgLoader.GetConf().File.MaxFileSizeBytes))
		}

		w.isWrittenFullTip = isFull

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

	if err := w.tryOpenNewFile(); err != nil && w.cfg.OnLogErr != nil {
		w.cfg.OnLogErr(err)
	}

	dealExpiredFilesTk := time.NewTicker(time.Minute * 10)
	defer dealExpiredFilesTk.Stop()
	for {
		select {
		case buf := <-w.bufCh:
			if err := doWriteMoreAsPossible(buf); err != nil && w.cfg.OnLogErr != nil {
				w.cfg.OnLogErr(err)
			}
		case <-w.flushSignCh:
			if err := doWriteMoreAsPossible([]byte{}); err != nil {
				w.finishFlush(err)
				break
			}
			if err := w.fp.Sync(); err != nil {
				w.finishFlush(err)
				break
			}
			w.finishFlush(nil)
		case <-dealExpiredFilesTk.C:
			go w.hdlExpiredFiles()
		}
	}
}
