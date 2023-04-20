##特点:
异步/高性能/因为磁盘IO繁忙阻塞主线程/适合并发输出日志量巨大的系统,避免日志写入阻塞导致性能下降(比如本司系统一小时最少20G日志)

## 示例
````
采用文件写入作为日志驱动：

import (
	"context"
	"github.com/995933447/log-go"
	"github.com/995933447/log-go/impl/loggerwriter"
	simpletracectx "github.com/995933447/simpletrace/context"
	"github.com/995933447/std-go/print"
	"testing"
)

func main() {
      writer := loggerwriter.NewFileLoggerWriter(
		"/var/log/testlog", // 日志目录
		10000, // 单文件最大大小,超出且未达到切换新文件事件则不再输出
		10, // 每十秒检查一次单个日志文件是否已满
		loggerwriter.OpenNewFileByByDateHour, // 根据小时进行日志文件切分
		100000, // 日志缓冲大小,日志写入线程写入文件阻塞时候,日志内容会在缓冲区先保存起来
	)

    // 日志写入文件的同时会输出屏幕上
	writer.EnableStdoutPrinter()
	go func() {
		if err := writer.Loop(); err != nil {
			panic(err)
		}
	}()

	logger := log.NewLogger(writer)
	ctx := simpletracectx.New("testlog", context.TODO(),"", "")
	logger.Debugf(ctx, "err:%v", "unknown err.")
	logger.Errorf(nil, "err:%v", "unknown err.")
	
	// 日志写入文件的同时不输出屏幕上
	writer.DisableStdoutPrinter()
	logger.Error(ctx, "err one")

	if err := logger.Flush(); err != nil {
		t.Fatal(err)
	}
	t.Log("finish.")
}
````

### logger 支持的方法
func NewLogger(loggerWriter LoggerWriter) *Logger

func (l *Logger) GetWriter() LoggerWriter

func (l *Logger) SetLogLevel(level Level) 

func (l *Logger) GetLogLevel() Level

func (l *Logger) Debug(ctx context.Context, content interface{})

func (l *Logger) Info(ctx context.Context, content interface{}) 

func (l *Logger) Warn(ctx context.Context, content interface{})

func (l *Logger) Error(ctx context.Context, content interface{})

func (l *Logger) Fatal(ctx context.Context, content interface{})

func (l *Logger) Panic(ctx context.Context, content interface{}) 

func (l *Logger) Debugf(ctx context.Context, format string, args ...interface{})

func (l *Logger) Infof(ctx context.Context, format string, args ...interface{}) 

func (l *Logger) Warnf(ctx context.Context, format string, args ...interface{}) 

func (l *Logger) Errorf(ctx context.Context, format string, args ...interface{})

func (l *Logger) Fatalf(ctx context.Context, format string, args ...interface{}) 

func (l *Logger) Panicf(ctx context.Context, format string, args ...interface{})

func (l *Logger) Flush()