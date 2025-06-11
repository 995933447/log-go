package logger

import "fmt"

type Color int

const (
	ColorNil Color = iota
	ColorRed
	ColorGreen
	ColorLightGreen
	ColorYellow
	ColorBlue
	ColorPurple
)

var ColorToStdoutMap = map[Color]string{
	ColorNil:        fmt.Sprintf("\u001B[0m"),
	ColorRed:        fmt.Sprintf("\x1b[%dm", 91),
	ColorLightGreen: fmt.Sprintf("\x1b[%dm", 92),
	ColorYellow:     fmt.Sprintf("\x1b[%dm", 93),
	ColorGreen:      fmt.Sprintf("\x1b[%dm", 33),
	ColorBlue:       fmt.Sprintf("\u001B[%d;1m", 36),
	ColorPurple:     fmt.Sprintf("\x1b[%dm", 95),
}

func GetColorStdout(color Color) (string, error) {
	if stdout, ok := ColorToStdoutMap[color]; ok {
		return stdout, nil
	}
	return "", fmt.Errorf("not support color type %d", color)
}
