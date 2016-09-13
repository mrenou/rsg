package inputs

import (
	"rsg/loggers"
	"rsg/utils"
	"rsg/consts"
)

func QueryContinue() {
	loggers.Print(loggers.Info, "Press to continue...")
	for _, _ = range consts.LINE_BREAK {
		_, err := StdinReader.ReadByte()
		utils.ExitIfError(err)
	}

}
