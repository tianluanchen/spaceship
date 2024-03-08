package pkg

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// log level
const (
	LDEBUG = iota + 1 // 1
	LINFO             // 2
	LWARN             // 3
	LERROR            // 4
	LFATAL            // 5
)

var logMutex sync.Mutex
var logOuput io.Writer = os.Stderr
var logLevel int = LDEBUG

func doWithLock(f func()) {
	logMutex.Lock()
	defer logMutex.Unlock()
	f()
}

// if level >= logLevel, continue
func checkLogLevel(level int) bool {
	return level >= logLevel
}

func SetLogOutput(w io.Writer) {
	doWithLock(func() {
		logOuput = w
	})
}

// use LDEBUG, LINFO, LWARN, LERROR, LFATAL
func SetLogLevel(level int) {
	doWithLock(func() {
		logLevel = level
	})
}

type Logger struct {
	prefix string
}

func NewLogger(prefix ...string) *Logger {
	l := &Logger{}
	if len(prefix) > 0 {
		l.prefix = prefix[0]
	}
	return l
}

// if format is not empty, similar to fmt.Printf
//
// if format is empty and ln is false, similar to fmt.Print
//
// if format is empty and ln is true, similar to fmt.Println
func (l *Logger) write(level int, format string, ln bool, v ...any) {
	if !checkLogLevel(level) {
		return
	}
	buf := &strings.Builder{}
	buf.WriteString(time.Now().Format("2006-01-02T15:04:05.000Z0700  "))

	switch level {
	case LDEBUG:
		buf.WriteString("DEBUG")
	case LINFO:
		buf.WriteString("INFO")
	case LWARN:
		buf.WriteString("WARN")
	case LERROR:
		buf.WriteString("ERROR")
	case LFATAL:
		buf.WriteString("FATAL")
	default:
		buf.WriteString("UNKNOWN")
	}
	buf.WriteString("  ")
	_, file, line, _ := runtime.Caller(2)
	// pc, file, line, _ := runtime.Caller(2)
	// funcName := runtime.FuncForPC(pc).Name()
	index := strings.LastIndexByte(file, '/')
	if index > -1 {
		file = file[index+1:]
	}
	buf.WriteString(file + ":" + strconv.Itoa(line) + "  ")
	buf.WriteString(l.prefix)
	if format == "" {
		if ln {
			fmt.Fprintln(buf, v...)
		} else {
			fmt.Fprint(buf, v...)
			buf.WriteByte('\n')
		}
	} else {
		fmt.Fprintf(buf, format, v...)
		buf.WriteByte('\n')
	}
	doWithLock(func() {
		fmt.Fprint(logOuput, buf.String())
	})
}

func (l *Logger) Debug(v ...interface{}) {
	l.write(LDEBUG, "", false, v...)
}
func (l *Logger) Debugln(v ...interface{}) {
	l.write(LDEBUG, "", true, v...)
}
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.write(LDEBUG, format, false, v...)
}
func (l *Logger) Info(v ...interface{}) {
	l.write(LINFO, "", false, v...)
}
func (l *Logger) Infoln(v ...interface{}) {
	l.write(LINFO, "", true, v...)
}
func (l *Logger) Infof(format string, v ...interface{}) {
	l.write(LINFO, format, false, v...)
}

func (l *Logger) Warn(v ...interface{}) {
	l.write(LWARN, "", false, v...)
}
func (l *Logger) Warnln(v ...interface{}) {
	l.write(LWARN, "", true, v...)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.write(LWARN, format, false, v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.write(LERROR, "", false, v...)
}
func (l *Logger) Errorln(v ...interface{}) {
	l.write(LERROR, "", true, v...)
}
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.write(LERROR, format, false, v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.write(LFATAL, "", false, v...)
	os.Exit(1)
}
func (l *Logger) Fatalln(v ...interface{}) {
	l.write(LFATAL, "", true, v...)
	os.Exit(1)
}
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.write(LFATAL, format, false, v...)
	os.Exit(1)
}
