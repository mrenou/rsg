package loggers

import (
	"os"
	"io"
	"fmt"
)

const (
	Verbose Level = iota
	OptionalInfo = iota
	Info = iota
	Warning = iota
	Error = iota
)

type Level int

var (
	VerboseFlag bool = false
	OptionalInfoFlag bool = true
	verboseWriter io.Writer
	optionalInfoWriter io.Writer
	infoWriter io.Writer
	warningWriter io.Writer
	errorWriter io.Writer
)

func InitDefaultLog() {
	InitLog(os.Stdout, os.Stdout, os.Stdout, os.Stdout, os.Stderr)
}

func InitLog(pVerboseWriter, pOptionalInfoWriter, pInfoWriter, pWarningWriter, pErrorWriter io.Writer) {
	verboseWriter = pVerboseWriter
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
	if level == Verbose {
		if VerboseFlag == false {
			return
		}
		writer = verboseWriter
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
