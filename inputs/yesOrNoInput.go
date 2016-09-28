package inputs

import (
	"strings"
	"rsg/outputs"
	"rsg/utils"
	"rsg/consts"
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
		outputs.Printf(outputs.Info, "%s[%s/%s] ", query, yes, no)
		resp, err := StdinReader.ReadString(consts.LINE_BREAK_LAST_CHAR)
		utils.ExitIfError(err)
		resp = strings.TrimSpace(resp)
		resp = strings.TrimSuffix(resp, consts.LINE_BREAK)
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
