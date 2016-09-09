package loggers

import (
	"os"
	"io"
	"fmt"
)

const (
	Debug Level = iota
	OptionalInfo = iota
	Info = iota
	Warning = iota
	Error = iota
)

type Level int

var (
	DebugFlag bool = false
	OptionalInfoFlag bool = true
	debugWriter io.Writer
	optionalInfoWriter io.Writer
	infoWriter io.Writer
	warningWriter io.Writer
	errorWriter io.Writer
)

func InitDefaultLog() {
	InitLog(os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stderr)
}

func InitLog(pDebugWriter, pOptionalInfoWriter, pInfoWriter, pWarningWriter, pErrorWriter io.Writer) {
	debugWriter = pDebugWriter
	optionalInfoWriter = pOptionalInfoWriter
	infoWriter = pInfoWriter
	warningWriter = pWarningWriter
	errorWriter = pErrorWriter
}

func Printf(level Level, format string, v ...interface{}) {
	print(level, fmt.Sprintf(format, v...))
}

func Print(level Level, v ...interface{}) {
	print(level, fmt.Sprint(v...))
}

func print(level Level, toPrint string) {
	var writer io.Writer;
	if level == Debug {
		if DebugFlag == false {
			return
		}
		writer = debugWriter
		toPrint = "DEBUG: " + toPrint;
	}
	if level == OptionalInfo {
		if OptionalInfoFlag == false {
			return
		}
		writer = optionalInfoWriter
	}
	if level == Info {
		writer = infoWriter
	}
	if level == Warning {
		writer = warningWriter
		toPrint = "WARNING: " + toPrint;
	}
	if level == Error {
		writer = errorWriter
		toPrint = "ERROR: " + toPrint;
	}
	fmt.Fprint(writer, toPrint)
}
