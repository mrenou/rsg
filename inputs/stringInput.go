package inputs

import (
	"strings"
	"rsg/loggers"
	"rsg/utils"
)

func QueryString(query string) string {
	for {
		loggers.Printf(loggers.Info, "%v ", query)
		answer, err := StdinReader.ReadString('\n')
		utils.ExitIfError(err)
		answer = strings.TrimSpace(answer)
		answer = strings.TrimSuffix(answer, "\n")
		if (answer != "") {
			return answer
		}
	}
}
