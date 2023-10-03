package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

type Level int

const calldepth = 3

const (
	LvPanic Level = iota
	LvFatal
	LvError
	LvWarn
	LvInfo
	LvDebug

	LvMax
)

type OutputFn func(calldepth int, s string) error

func dropOutputFn(calldepth int, s string) error {
	return nil
}

type DefaultLogger struct {
	*log.Logger
	out [LvMax]OutputFn
}

func NewDefaultLogger(out io.Writer, prefix string, flag int, lv Level) *DefaultLogger {
	level := lv

	// 从环境变量读取覆盖level
	lvStr := os.Getenv("RPCX_LOG_LEVEL")
	if lvStr != "" {
		if lv, err := strconv.Atoi(lvStr); err == nil {
			level = Level(lv)
		}
	}

	dl := new(DefaultLogger)
	dl.Logger = log.New(out, prefix, flag)

	// panic，fatal 默认一定会输出日志
	dl.out[LvPanic] = dl.Output
	dl.out[LvFatal] = dl.Output

	for i := LvError; i < LvMax; i++ {
		if i <= level {
			dl.out[i] = dl.Output
		} else {
			dl.out[i] = dropOutputFn
		}
	}

	return dl
}

func (l *DefaultLogger) Debug(v ...interface{}) {
	_ = l.out[LvDebug](calldepth, header("DEBUG", fmt.Sprint(v...)))
}

func (l *DefaultLogger) Debugf(format string, v ...interface{}) {
	_ = l.out[LvDebug](calldepth, header("DEBUG", fmt.Sprintf(format, v...)))
}

func (l *DefaultLogger) Info(v ...interface{}) {
	_ = l.out[LvInfo](calldepth, header("INFO", fmt.Sprint(v...)))
}

func (l *DefaultLogger) Infof(format string, v ...interface{}) {
	_ = l.out[LvInfo](calldepth, header("INFO", fmt.Sprintf(format, v...)))
}

func (l *DefaultLogger) Warn(v ...interface{}) {
	_ = l.out[LvWarn](calldepth, header("WARN", fmt.Sprint(v...)))
}

func (l *DefaultLogger) Warnf(format string, v ...interface{}) {
	_ = l.out[LvWarn](calldepth, header("WARN", fmt.Sprintf(format, v...)))
}

func (l *DefaultLogger) Error(v ...interface{}) {
	_ = l.out[LvError](calldepth, header("ERROR", fmt.Sprint(v...)))
}

func (l *DefaultLogger) Errorf(format string, v ...interface{}) {
	_ = l.out[LvError](calldepth, header("ERROR", fmt.Sprintf(format, v...)))
}

func (l *DefaultLogger) Fatal(v ...interface{}) {
	_ = l.out[LvFatal](calldepth, header("FATAL", fmt.Sprint(v...)))
	os.Exit(1)
}

func (l *DefaultLogger) Fatalf(format string, v ...interface{}) {
	_ = l.out[LvFatal](calldepth, header("FATAL", fmt.Sprintf(format, v...)))
	os.Exit(1)
}

func (l *DefaultLogger) Panic(v ...interface{}) {
	_ = l.out[LvPanic](calldepth, header("PANIC", fmt.Sprint(v...)))
	panic(fmt.Sprint(v...))
}

func (l *DefaultLogger) Panicf(format string, v ...interface{}) {
	_ = l.out[LvPanic](calldepth, header("PANIC", fmt.Sprintf(format, v...)))
	panic(fmt.Sprintf(format, v...))
}

func header(lvl, msg string) string {
	return fmt.Sprintf("%s: %s", lvl, msg)
}
