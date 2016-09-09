package inputs

import (
	"rsg/loggers"
	"rsg/utils"
)

func QueryContinue() {
	loggers.Print(loggers.Info, "press to continue...")
	_, err := StdinReader.ReadByte()
	utils.ExitIfError(err)
}
