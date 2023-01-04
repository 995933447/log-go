package log

import (
	"context"
	"fmt"
	"github.com/995933447/std-go/print"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelPanic
	LevelFatal
)

var levelToStrMap = map[Level]string{
	LevelDebug: "DBG",
	LevelInfo: "INFO",
	LevelWarn: "WARN",
	LevelError: "ERR",
	LevelFatal: "FATAL",
	LevelPanic: "PANIC",
}

type Formatter interface {
	Sprintf(ctx context.Context, level Level, stdoutColor print.Color, format string, args ...interface{}) (string, error)
}

func TransferLevelToStr(level Level) (string, error) {
	if str, ok := levelToStrMap[level]; ok {
		return str, nil
	} else {
		return "", fmt.Errorf("unknow level %d", level)
	}
}

