package inputs

import (
	"strings"
	"rsg/loggers"
	"rsg/utils"
)

func QueryYesOrNo(query string, defaultAnswer bool) bool {
	for {
		yes := "y"
		no := "n"
		if defaultAnswer {
			yes = strings.ToUpper(yes)
		} else {
			no = strings.ToUpper(no)
		}
		loggers.Printf(loggers.Info, "%s[%s/%s] ", query, yes, no)
		resp, err := StdinReader.ReadString('\n')
		utils.ExitIfError(err)
		resp = strings.TrimSuffix(resp, "\n")
		resp = strings.ToLower(resp)
		switch resp {
		case "y":
			return true
		case "n":
			return false
		case "":
			return defaultAnswer
		}
	}
}
