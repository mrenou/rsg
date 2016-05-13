package loggers

import (
	"os"
	"log"
	"io"
)

var (
	debugFlag bool = true
	debug   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

func InitDefaultLog() {
	InitLog(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
}

func InitLog(debugWriter, infoWriter, warningWriter, errorWriter io.Writer) {
	debug = log.New(debugWriter, "DEBUG: ", 0)
	Info = log.New(infoWriter, "", 0)
	Warning = log.New(warningWriter, "WARNING: ", 0)
	Error = log.New(errorWriter, "ERROR: ", 0)
}

func DebugPrintf(format string, v ...interface{}) {
	if (debugFlag) {
		debug.Printf(format, v...)
	}
}
