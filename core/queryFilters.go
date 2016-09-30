package core

import (
	"rsg/outputs"
	"rsg/inputs"
	"strings"
)

func QueryFiltersIfNecessary(restorationContext *RestorationContext) {
	if len(restorationContext.Options.Filters) == 0 && outputs.OptionalInfoFlag == true {
		if inputs.QueryYesOrNo("Do you want add filter(s) on files to retrieves ?", false) {
			filtersAsString := inputs.QueryString("Write filters separated by '|'. You can use global * and ?:")
			restorationContext.Options.Filters = strings.Split(filtersAsString, "|")
		}
	}
}