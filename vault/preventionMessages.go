package vault

import (
	"rsg/awsutils"
	"rsg/loggers"
	"rsg/inputs"
	"rsg/options"
)

func DisplayInfoAboutCosts(options options.Options) {
	if options.InfoMessage {
		loggers.Printf(loggers.OptionalInfo, "###################################################################################\n")
		loggers.Printf(loggers.OptionalInfo, "The use of Amazone Web Service Glacier could generate additional costs.\n")
		loggers.Printf(loggers.OptionalInfo, "The author(s) of this program cannot be held responsible for these additional costs\n")
		loggers.Printf(loggers.OptionalInfo, "More information about pricing : https://aws.amazon.com/glacier/pricing/\n")
		loggers.Printf(loggers.OptionalInfo, "####################################################################################\n")
		inputs.QueryContinue()
	}
}

func DisplayWarnIfNotFreeTier(restorationContext *awsutils.RestorationContext) {
	if restorationContext.Options.InfoMessage {
		strategy := awsutils.GetDataRetrievalStrategy(restorationContext)
		if strategy != "FreeTier" {
			loggers.Printf(loggers.OptionalInfo, "##################################################################################################################\n")
			loggers.Printf(loggers.OptionalInfo, "Your data retrieval strategy is \"%v\", the retrieval operations could generate additional costs !!!\n", strategy)
			loggers.Printf(loggers.OptionalInfo, "Select strategy \"FreeTier\" to avoid these costs :\n")
			loggers.Printf(loggers.OptionalInfo, "http://docs.aws.amazon.com/amazonglacier/latest/dev/data-retrieval-policy.html#data-retrieval-policy-using-console\n")
			loggers.Printf(loggers.OptionalInfo, "##################################################################################################################\n")
			inputs.QueryContinue()
		}
	}

}