package inputs

import (
	"strings"
	"rsg/loggers"
	"rsg/utils"
)

func QueryString(query string) string {
	loggers.Print(loggers.Info, query)
	answer, err := StdinReader.ReadString('\n')
	utils.ExitIfError(err)
	return strings.TrimSuffix(answer, "\n")
}
