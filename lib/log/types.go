package log

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type LogFormatFunc func(flag string, level string, str string) string

func logFormatFuncDefault(flag string, level string, str string) string {
	return fmt.Sprintf("[%s] [%s]\t%s: %s", time.Now().Format("2006-01-02 15:04:05 UTC-07"), level, flag, str)
}

type Logger struct {
	locker    sync.RWMutex
	out       io.Writer
	formatter LogFormatFunc
	debug     bool
}
