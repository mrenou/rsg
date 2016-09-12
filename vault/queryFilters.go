package vault

import (
	"rsg/options"
	"rsg/loggers"
	"rsg/inputs"
	"strings"
	"rsg/awsutils"
)

func QueryFiltersIfNecessary(restorationContext *awsutils.RestorationContext, options options.Options) {
	if len(restorationContext.Options.Filters) == 0 && (!options.List || loggers.OptionalInfoFlag == true) {
		if inputs.QueryYesOrNo("Do you want add filter(s) on files to retrieves ?", false) {
			filtersAsString := inputs.QueryString("Write filters separated by '|'. You can use global * and ?:")
			restorationContext.Options.Filters = strings.Split(filtersAsString, "|")
		}
	}
}