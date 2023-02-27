package log

import (
	"io"
	"os"
	"sync"
)

func NewLogger(output io.Writer, formatter LogFormatFunc) *Logger {
	l := &Logger{
		locker: sync.RWMutex{},
	}
	if output == nil {
		l.out = os.Stdout
	} else {
		l.out = output
	}
	if formatter == nil {
		l.formatter = logFormatFuncDefault
	} else {
		l.formatter = formatter
	}
	return l
}

func (l *Logger) SetOutput(output io.Writer) {
	if output == nil {
		return
	}
	l.locker.Lock()
	defer l.locker.Unlock()
	l.out = output
}

func (l *Logger) SetLogFormatFunc(f LogFormatFunc) {
	if f == nil {
		return
	}
	l.locker.Lock()
	defer l.locker.Unlock()
	l.formatter = f
}

func (l *Logger) SetDebug(debug bool) {
	l.locker.Lock()
	defer l.locker.Unlock()
	l.debug = debug
}

func (l *Logger) print(flag string, level string, str string) {
	l.locker.RLock()
	defer l.locker.RUnlock()
	s := l.formatter(flag, level, str)
	if s[len(s)-1] != '\n' {
		s += "\n"
	}
	l.out.Write([]byte(s))
}

func (l *Logger) Info(flag string, str string) {
	l.print(flag, "Info", str)
}

func (l *Logger) Warn(flag string, str string) {
	l.print(flag, "Warn", str)
}

func (l *Logger) Error(flag string, str string) {
	l.print(flag, "Error", str)
}

func (l *Logger) Fatal(flag string, str string) {
	l.print(flag, "Fatal", str)
}

func (l *Logger) Debug(flag string, str string) {
	if !l.debug {
		return
	}
	l.print(flag, "Debug", str)
}
