package logger

import (
	"log"
	"os"
)

var (
	InfoLogger  *log.Logger
	WarnLogger  *log.Logger
	ErrorLogger *log.Logger
)

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarnLogger = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func Info(msg string, v ...interface{}) {
	InfoLogger.Printf(msg, v...)
}

func Warn(msg string, v ...interface{}) {
	WarnLogger.Printf(msg, v...)
}

func Error(msg string, err error, v ...interface{}) {
	if err != nil {
		ErrorLogger.Printf(msg+": %v", append(v, err)...)
	} else {
		ErrorLogger.Printf(msg, v...)
	}
}
