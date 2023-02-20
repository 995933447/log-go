package test

import (
	"context"
	"github.com/995933447/log-go"
	"github.com/995933447/log-go/impl/loggerwriter"
	simpletracectx "github.com/995933447/simpletrace/context"
	"github.com/995933447/std-go/print"
	"testing"
)

func TestStdoutLog(t *testing.T) {
	log.NewLogger(loggerwriter.NewStdoutLoggerWriter(print.ColorGreen)).Debugf(context.Background(), "err:%v", "unknown err.")
}

func TestFileLog(t *testing.T) {
	writer := loggerwriter.NewFileLoggerWriter(
		"/var/log/testlog",
		10000,
		10,
		loggerwriter.OpenNewFileByByDateHour,
		100000,
	)

	writer.EnableStdoutPrinter()
	go func() {
		if err := writer.Loop(); err != nil {
			panic(err)
		}
	}()

	logger := log.NewLogger(writer)
	ctx := simpletracectx.New("testlog", context.TODO(),"", "")
	logger.Debugf(ctx, "err:%v", "unknown err.")
	logger.Errorf(ctx, "err:%v", "unknown err.")
	writer.DisableStdoutPrinter()
	logger.Error(ctx, "err one")

	if err := logger.Flush(); err != nil {
		t.Fatal(err)
	}
	t.Log("finish.")
}
