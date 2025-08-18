package loggo

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/995933447/log-go/v2/loggo/logger"
	"github.com/995933447/log-go/v2/loggo/logger/writer"
	"github.com/995933447/runtimeutil"
	jsoniter "github.com/json-iterator/go"
)

var (
	moduleName                     = "loggo"
	nodeId                         = os.Getpid()
	defaultCfgLoader               *logger.ConfLoader
	defaultLogger, exceptionLogger *logger.Logger
)

func SetNodeId(n int) {
	nodeId = n
}

func SetModuleName(m string) {
	moduleName = m
}

func InitDefaultLogger(alertFunc writer.AlertFunc) error {
	cfgLoader := MustDefaultCfgLoader()
	cfg := cfgLoader.GetConf()
	var err error
	defaultLogger, err = InitWithAlertFileLogger(cfg.File.DefaultLogDir, moduleName, 6, cfgLoader, alertFunc)
	if err != nil {
		return err
	}
	return nil
}

func InitExceptionLogger() error {
	cfgLoader := MustDefaultCfgLoader()
	cfg := cfgLoader.GetConf()
	var err error
	exceptionLogger, err = InitFileLogger(cfg.File.ExceptionLogDir, "error", 5, cfgLoader)
	if err != nil {
		return err
	}
	return nil
}

func InitDefaultCfgLoader(cfgFile string, defaultLogCfg *logger.LogConf) error {
	var err error
	defaultCfgLoader, err = logger.NewConfLoader(cfgFile, 10, defaultLogCfg)
	if err != nil {
		return err
	}

	return nil
}

func MustDefaultCfgLoader() *logger.ConfLoader {
	if defaultCfgLoader == nil {
		panic("defaultCfgLoader not init")
	}
	return defaultCfgLoader
}

func GetDefaultCfgLoader() (*logger.ConfLoader, bool) {
	if defaultCfgLoader == nil {
		return nil, false
	}

	return defaultCfgLoader, true
}

func InitFileLogger(baseDir, filePrefix string, skipCall int, cfgLoader *logger.ConfLoader) (*logger.Logger, error) {
	writerCfg := &writer.FileWriterConf{
		ModuleName:               moduleName,
		FilePrefix:               filePrefix,
		BaseDir:                  baseDir,
		SkipCall:                 skipCall,
		LogCfgLoader:             cfgLoader,
		CheckFileFullIntervalSec: 10,
		BufChanLen:               100000,
		CheckTimeToOpenNewFile:   OpenNewFileByByDateHour,
		OnLogErr: func(err error) {
			fmt.Println(err)
		},
	}
	fileWriter, err := writer.NewFileWriter(writerCfg)
	if err != nil {
		return nil, err
	}
	runtimeutil.Go(func() {
		fileWriter.Loop()
	})
	var fileLogger *logger.Logger
	fileLogger = logger.NewLogger(fileWriter)
	return fileLogger, nil
}

func InitWithAlertFileLogger(baseDir, filePrefix string, skipCall int, cfgLoader *logger.ConfLoader, alertFunc writer.AlertFunc) (*logger.Logger, error) {
	writerCfg := &writer.FileWriterConf{
		ModuleName:               moduleName,
		FilePrefix:               filePrefix,
		BaseDir:                  baseDir,
		SkipCall:                 skipCall,
		LogCfgLoader:             cfgLoader,
		CheckFileFullIntervalSec: 10,
		BufChanLen:               100000,
		CheckTimeToOpenNewFile:   OpenNewFileByByDateHour,
		OnLogErr: func(err error) {
			fmt.Println(err)
		},
	}
	fileWriter, err := writer.NewFileWriter(writerCfg)
	if err != nil {
		return nil, err
	}
	runtimeutil.Go(func() {
		fileWriter.Loop()
	})
	var withAlertLogger *logger.Logger
	withAlertLogger = logger.NewLogger(writer.NewWithAlertWriter(fileWriter, cfgLoader, alertFunc))
	return withAlertLogger, nil
}

func OpenNewFileByByDateHour(writer *writer.FileWriter, lastOpenFileTime *time.Time, isNeverOpenFile bool) (string, bool) {
	fileName := writer.GetFilePrefix() + time.Now().Format("200601021504") + fmt.Sprintf("_%d", nodeId) + ".txt"

	if isNeverOpenFile {
		return fileName, true
	}

	if lastOpenFileTime.Hour() != time.Now().Hour() {
		return fileName, true
	}

	if writer.GetFileConf().MaxFileSizeBytes > 0 && writer.GetFileSize() >= writer.GetFileConf().MaxFileSizeBytes {
		return fileName, true
	}

	return "", false
}

func OnExit() {
	if defaultLogger != nil {
		if err := defaultLogger.Flush(); err != nil {
			fmt.Println("flush default logger, err:", err)
		}
	}

	for billName, billLogger := range billLoggerFactory.loggerMap {
		if err := billLogger.Flush(); err != nil {
			fmt.Println("flush bill logger:", billName, "err:", err)
		}
	}
}

func SetLogConfig(cfg *logger.LogConf) {
	if cfgLoader, ok := GetDefaultCfgLoader(); ok {
		cfgLoader.SetDefaultLogConf(cfg)
	}
}

func GetLevel() logger.Level {
	if cfgLoader, ok := GetDefaultCfgLoader(); ok {
		return cfgLoader.GetConf().File.GetLevel()
	}

	return logger.LevelDebug
}

type MsgFormatForPrint struct {
	args []interface{}
}

func (m *MsgFormatForPrint) String() string {
	var b strings.Builder
	b.Grow(15 * len(m.args))
	for i, a := range m.args {
		if i > 0 {
			b.WriteString(" - ")
		}
		switch x := a.(type) {
		case string:
			b.WriteString(x)
		case int8:
			b.WriteString(strconv.FormatInt(int64(x), 10))
		case int:
			b.WriteString(strconv.FormatInt(int64(x), 10))
		case int32:
			b.WriteString(strconv.FormatInt(int64(x), 10))
		case int64:
			b.WriteString(strconv.FormatInt(x, 10))
		case uint:
			b.WriteString(strconv.FormatUint(uint64(x), 10))
		case uint8:
			b.WriteString(strconv.FormatUint(uint64(x), 10))
		case uint32:
			b.WriteString(strconv.FormatUint(uint64(x), 10))
		case uint64:
			b.WriteString(strconv.FormatUint(x, 10))
		case float32:
			b.WriteString(strconv.FormatFloat(float64(x), 'f', -1, 32))
		case float64:
			b.WriteString(strconv.FormatFloat(x, 'f', -1, 64))
		case bool:
			b.WriteString(strconv.FormatBool(x))
		case *int8:
			b.WriteString(strconv.FormatInt(int64(*x), 10))
		case *int:
			b.WriteString(strconv.FormatInt(int64(*x), 10))
		case *int32:
			b.WriteString(strconv.FormatInt(int64(*x), 10))
		case *int64:
			b.WriteString(strconv.FormatInt(*x, 10))
		case *uint:
			b.WriteString(strconv.FormatUint(uint64(*x), 10))
		case *uint8:
			b.WriteString(strconv.FormatUint(uint64(*x), 10))
		case *uint32:
			b.WriteString(strconv.FormatUint(uint64(*x), 10))
		case *uint64:
			b.WriteString(strconv.FormatUint(*x, 10))
		case *float32:
			b.WriteString(strconv.FormatFloat(float64(*x), 'f', -1, 32))
		case *float64:
			b.WriteString(strconv.FormatFloat(*x, 'f', -1, 64))
		case *bool:
			b.WriteString(strconv.FormatBool(*x))
		case error:
			b.WriteString(x.Error())
		case fmt.Stringer:
			b.WriteString(x.String())
		default:
			if j, _ := jsoniter.ConfigFastest.MarshalToString(x); j != "" {
				b.WriteString(j)
			}
		}
	}

	return b.String()
}

func fmtMsgForPrint(v ...interface{}) *MsgFormatForPrint {
	return &MsgFormatForPrint{args: v}
}

func Debug(content interface{}) {
	defaultLogger.Debug(content)
}

func Info(content interface{}) {
	defaultLogger.Info(content)
}

func Warn(content interface{}) {
	defaultLogger.Warn(content)
}

func Important(content interface{}) {
	defaultLogger.Important(content)
}

func Error(content interface{}) {
	defaultLogger.Error(content)
}

func Panic(content interface{}) {
	defaultLogger.Panic(content)
}

func Fatal(content interface{}) {
	defaultLogger.Fatal(content)
}

func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

func Importantf(format string, args ...interface{}) {
	defaultLogger.Importantf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	defaultLogger.Panicf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}

func PrintDebug(args ...interface{}) {
	defaultLogger.Debug(fmtMsgForPrint(args...))
}

func PrintInfo(args ...interface{}) {
	defaultLogger.Info(fmtMsgForPrint(args...))
}

func PrintImportant(args ...interface{}) {
	defaultLogger.Important(fmtMsgForPrint(args...))
}

func PrintWarn(args ...interface{}) {
	defaultLogger.Warn(fmtMsgForPrint(args...))
}

func PrintError(args ...interface{}) {
	defaultLogger.Error(fmtMsgForPrint(args...))
}

func PrintPanic(args ...interface{}) {
	defaultLogger.Panic(fmtMsgForPrint(args...))
}

func PrintFatal(args ...interface{}) {
	defaultLogger.Fatal(fmtMsgForPrint(args...))
}

func WriteBySkipCall(level logger.Level, skipCall int, format string, args ...interface{}) {
	if err := defaultLogger.WriteBySkipCall(level, skipCall, append([]interface{}{format}, args...)...); err != nil {
		fmt.Println(err)
	}
}

func EnableStdoutPrinter() {
	defaultLogger.EnableStdoutPrinter()
}

func DisableStdoutPrinter() {
	defaultLogger.DisableStdoutPrinter()
}
