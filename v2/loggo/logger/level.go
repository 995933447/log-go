package logger

import (
	"fmt"
)

type Level int

const (
	LevelDebug = iota
	LevelInfo
	LevelImportant
	LevelWarn
	LevelError
	LevelPanic // panic级别
	LevelFatal // 该级别panic并且退出
)

var LevelToStrMap = map[Level]string{
	LevelDebug:     "DBG",
	LevelInfo:      "INFO",
	LevelImportant: "IMP",
	LevelWarn:      "WARN",
	LevelError:     "ERR",
	LevelPanic:     "PANIC",
	LevelFatal:     "FATAL",
}

func TransferLevelToStr(level Level) (string, error) {
	if str, ok := LevelToStrMap[level]; ok {
		return str, nil
	} else {
		return "", fmt.Errorf("unknow level %d", level)
	}
}

func TransStrToLevel(levelStr string) Level {
	var level Level
	for l, name := range LevelToStrMap {
		if name == levelStr {
			level = l
			break
		}
	}
	return level
}
