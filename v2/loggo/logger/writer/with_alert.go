package writer

import (
	logger2 "github.com/995933447/log-go/v2/loggo/logger"
)

func NewWithAlertWriter(realWriter logger2.Writer, cfgLoader *logger2.ConfLoader, alertFunc AlertFunc) *WithAlertWriter {
	return &WithAlertWriter{
		cfgLoader:  cfgLoader,
		alertFunc:  alertFunc,
		realWriter: realWriter,
	}
}

var _ logger2.Writer = (*WithAlertWriter)(nil)

type AlertFunc func(msg *logger2.Msg)

type WithAlertWriter struct {
	realWriter logger2.Writer
	cfgLoader  *logger2.ConfLoader
	alertFunc  AlertFunc
}

func (w *WithAlertWriter) GetAlertLevel() logger2.Level {
	levelStr := w.cfgLoader.GetConf().AlertLevel
	if levelStr == "" {
		return logger2.LevelWarn
	}
	return logger2.TransStrToLevel(levelStr)
}

func (w *WithAlertWriter) WriteMsg(msg *logger2.Msg) error {
	if err := w.realWriter.WriteMsg(msg); err != nil {
		return err
	}

	if w.alertFunc == nil {
		return nil
	}

	if w.GetAlertLevel() > msg.Level {
		return nil
	}

	w.alertFunc(msg)

	return nil
}

func (w *WithAlertWriter) GetMsg(level logger2.Level, format string, args ...interface{}) (*logger2.Msg, error) {
	return w.realWriter.GetMsg(level, format, args...)
}

func (w *WithAlertWriter) GetMsgBySkipCall(level logger2.Level, skipCall int, format string, args ...interface{}) (*logger2.Msg, error) {
	return w.realWriter.GetMsgBySkipCall(level, skipCall, format, args...)
}

func (w *WithAlertWriter) GetSkipCall() int {
	return w.realWriter.GetSkipCall() - 1
}

func (w *WithAlertWriter) Write(level logger2.Level, format string, args ...interface{}) error {
	if w.alertFunc == nil {
		if err := w.realWriter.Write(level, format, args...); err != nil {
			return err
		}
		return nil
	}

	if w.GetAlertLevel() > level {
		if err := w.realWriter.Write(level, format, args...); err != nil {
			return err
		}
		return nil
	}

	msg, err := w.realWriter.GetMsg(level, format, args...)
	if err != nil {
		return err
	}

	if err = w.realWriter.WriteMsg(msg); err != nil {
		return err
	}

	w.alertFunc(msg)

	return nil
}

func (w *WithAlertWriter) WriteBySkipCall(level logger2.Level, skipCall int, format string, args ...interface{}) error {
	if w.alertFunc == nil {
		if err := w.realWriter.WriteBySkipCall(level, skipCall, format, args...); err != nil {
			return err
		}
		return nil
	}

	if w.GetAlertLevel() > level {
		if err := w.realWriter.WriteBySkipCall(level, skipCall, format, args...); err != nil {
			return err
		}
		return nil
	}

	msg, err := w.realWriter.GetMsgBySkipCall(level, skipCall, format, args...)
	if err != nil {
		return err
	}

	if err = w.realWriter.WriteMsg(msg); err != nil {
		return err
	}

	w.alertFunc(msg)

	return nil
}

func (w *WithAlertWriter) Flush() error {
	return w.realWriter.Flush()
}
