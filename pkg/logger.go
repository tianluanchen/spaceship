package pkg

import (
	"io"
	"log"
	"os"
	"sync"
)

// log level
const (
	lDEBUG = iota + 1
	lINFO
	lWARN
	lERROR
	lFATAL
)

type Logger struct {
	level       int
	output      io.Writer
	debugLogger *log.Logger
	warnLogger  *log.Logger
	infoLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
}

var logMutex sync.Mutex

func NewLoggerWithPrefix(w io.Writer, prefix string) *Logger {
	if w == nil {
		w = os.Stderr
	}
	flag := log.Ldate | log.Ltime | log.Lmsgprefix
	return &Logger{
		level:       lDEBUG,
		output:      w,
		debugLogger: log.New(w, "[DEBUG] "+prefix, flag),
		warnLogger:  log.New(w, "[WARN] "+prefix, flag),
		infoLogger:  log.New(w, "[INFO] "+prefix, flag),
		errorLogger: log.New(w, "[ERROR] "+prefix, flag),
		fatalLogger: log.New(w, "[FATAL] "+prefix, flag),
	}
}

func NewLogger(w io.Writer) *Logger {
	return NewLoggerWithPrefix(w, "")
}
func (l *Logger) SetDebugLevel() {
	l.level = lDEBUG
}
func (l *Logger) SetWarnLevel() {
	l.level = lWARN
}
func (l *Logger) SetInfoLevel() {
	l.level = lINFO
}
func (l *Logger) SetErrorLevel() {
	l.level = lERROR
}
func (l *Logger) SetFatalLevel() {
	l.level = lFATAL
}

func (l *Logger) Debug(v ...interface{}) {
	if l.level > lDEBUG {
		return
	}
	if l.output == os.Stderr || l.output == os.Stdout {
		logMutex.Lock()
		defer logMutex.Unlock()
	}
	l.debugLogger.Println(v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	if l.level > lDEBUG {
		return
	}
	if l.output == os.Stderr || l.output == os.Stdout {
		logMutex.Lock()
		defer logMutex.Unlock()
	}
	l.debugLogger.Printf(format, v...)
}

func (l *Logger) Info(v ...interface{}) {
	if l.level > lINFO {
		return
	}
	if l.output == os.Stderr || l.output == os.Stdout {
		logMutex.Lock()
		defer logMutex.Unlock()
	}
	l.infoLogger.Println(v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	if l.level > lINFO {
		return
	}
	if l.output == os.Stderr || l.output == os.Stdout {
		logMutex.Lock()
		defer logMutex.Unlock()
	}
	l.infoLogger.Printf(format, v...)
}

func (l *Logger) Warn(v ...interface{}) {
	if l.level > lWARN {
		return
	}
	if l.output == os.Stderr || l.output == os.Stdout {
		logMutex.Lock()
		defer logMutex.Unlock()
	}
	l.warnLogger.Println(v...)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	if l.level > lWARN {
		return
	}
	if l.output == os.Stderr || l.output == os.Stdout {
		logMutex.Lock()
		defer logMutex.Unlock()
	}
	l.warnLogger.Printf(format, v...)
}

func (l *Logger) Error(v ...interface{}) {
	if l.level > lERROR {
		return
	}
	if l.output == os.Stderr || l.output == os.Stdout {
		logMutex.Lock()
		defer logMutex.Unlock()
	}
	l.errorLogger.Println(v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	if l.level > lERROR {
		return
	}
	if l.output == os.Stderr || l.output == os.Stdout {
		logMutex.Lock()
		defer logMutex.Unlock()
	}
	l.errorLogger.Printf(format, v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	if l.level > lFATAL {
		return
	}
	if l.output == os.Stderr || l.output == os.Stdout {
		logMutex.Lock()
		defer logMutex.Unlock()
	}
	l.fatalLogger.Println(v...)
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	if l.level > lFATAL {
		return
	}
	if l.output == os.Stderr || l.output == os.Stdout {
		logMutex.Lock()
		defer logMutex.Unlock()
	}
	l.fatalLogger.Printf(format, v...)
	os.Exit(1)
}
