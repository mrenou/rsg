package core

import (
	"rsg/awsutils"
	"rsg/loggers"
	"rsg/inputs"
	"rsg/options"
)

func DisplayInfoAboutCosts(options options.Options) {
	if options.InfoMessage {
		loggers.Printfln(loggers.OptionalInfo, "###################################################################################")
		loggers.Printfln(loggers.OptionalInfo, "The use of Amazone Web Service Glacier could generate additional costs.")
		loggers.Printfln(loggers.OptionalInfo, "The author(s) of this program cannot be held responsible for these additional costs")
		loggers.Printfln(loggers.OptionalInfo, "More information about pricing : https://aws.amazon.com/glacier/pricing/")
		loggers.Printfln(loggers.OptionalInfo, "####################################################################################")
		inputs.QueryContinue()
	}
}

func DisplayWarnIfNotFreeTier(restorationContext *awsutils.RestorationContext) {
	if restorationContext.Options.InfoMessage {
		strategy := awsutils.GetDataRetrievalStrategy(restorationContext)
		if strategy != "FreeTier" {
			loggers.Printfln(loggers.OptionalInfo, "##################################################################################################################")
			loggers.Printfln(loggers.OptionalInfo, "Your data retrieval strategy is \"%v\", the retrieval operations could generate additional costs !!!", strategy)
			loggers.Printfln(loggers.OptionalInfo, "Select strategy \"FreeTier\" to avoid these costs :")
			loggers.Printfln(loggers.OptionalInfo, "http://docs.aws.amazon.com/amazonglacier/latest/dev/data-retrieval-policy.html#data-retrieval-policy-using-console")
			loggers.Printfln(loggers.OptionalInfo, "##################################################################################################################")
			inputs.QueryContinue()
		}
	}

}