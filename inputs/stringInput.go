package inputs

import (
	"strings"
	"rsg/outputs"
	"rsg/utils"
	"rsg/consts"
)

func QueryString(query string) string {
	for {
		outputs.Printf(outputs.Info, "%v ", query)
		answer, err := StdinReader.ReadString(consts.LINE_BREAK_LAST_CHAR)
		utils.ExitIfError(err)
		answer = strings.TrimSpace(answer)
		answer = strings.TrimSuffix(answer, consts.LINE_BREAK)
		if (answer != "") {
			return answer
		}
	}
}
