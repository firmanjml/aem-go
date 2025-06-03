package logger

import (
	"log"
	"os"
)

type Logger struct {
	debug  bool
	logger *log.Logger
}

func New(debug bool) *Logger {
	return &Logger{
		debug:  debug,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debug {
		l.logger.Printf("[DEBUG] "+format, args...)
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.logger.Printf("[INFO] "+format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.logger.Printf("[ERROR] "+format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.logger.Fatalf("[FATAL] "+format, args...)
}
