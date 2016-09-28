package core

import (
	"rsg/awsutils"
	"rsg/outputs"
	"rsg/inputs"
	"rsg/options"
)

func DisplayInfoAboutCosts(options options.Options) {
	if options.InfoMessage {
		outputs.Printfln(outputs.OptionalInfo, "###################################################################################")
		outputs.Printfln(outputs.OptionalInfo, "The use of Amazone Web Service Glacier could generate additional costs.")
		outputs.Printfln(outputs.OptionalInfo, "The author(s) of this program cannot be held responsible for these additional costs")
		outputs.Printfln(outputs.OptionalInfo, "More information about pricing : https://aws.amazon.com/glacier/pricing/")
		outputs.Printfln(outputs.OptionalInfo, "####################################################################################")
		inputs.QueryContinue()
	}
}

func DisplayWarnIfNotFreeTier(restorationContext *RestorationContext) {
	if restorationContext.Options.InfoMessage {
		strategy := awsutils.GetDataRetrievalStrategy(restorationContext.GlacierClient)
		if strategy != "FreeTier" {
			outputs.Printfln(outputs.OptionalInfo, "##################################################################################################################")
			outputs.Printfln(outputs.OptionalInfo, "Your data retrieval strategy is \"%v\", the retrieval operations could generate additional costs !!!", strategy)
			outputs.Printfln(outputs.OptionalInfo, "Select strategy \"FreeTier\" to avoid these costs :")
			outputs.Printfln(outputs.OptionalInfo, "http://docs.aws.amazon.com/amazonglacier/latest/dev/data-retrieval-policy.html#data-retrieval-policy-using-console")
			outputs.Printfln(outputs.OptionalInfo, "##################################################################################################################")
			inputs.QueryContinue()
		}
	}

}