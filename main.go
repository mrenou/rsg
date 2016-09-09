package main

import (
	"rsg/vault"
	"rsg/loggers"
	"rsg/awsutils"
	"rsg/utils"
	"rsg/options"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	loggers.InitDefaultLog()
	displayInfoAboutCosts()
	options := options.ParseOptions()
	session := awsutils.BuildSession(options.AwsId, options.AwsSecret)
	accountId, err := awsutils.GetAccountId(session)
	utils.ExitIfError(err)
	region, vaultName := vault.SelectRegionVault(accountId, session, options.Region, options.Vault)
	restorationContext := awsutils.CreateRestorationContext(session, accountId, region, vaultName, options)
	displayWarnIfNotFreeTier(restorationContext)
	vault.DownloadMappingArchive(restorationContext)
	if options.List {
		vault.ListArchives(restorationContext)
	} else {
		err = vault.CheckDestinationDirectory(restorationContext)
		utils.ExitIfError(err)
		vault.DownloadArchives(restorationContext)
	}
}


func displayInfoAboutCosts() {
	loggers.Printf(loggers.Info, "###################################################################################\n")
	loggers.Printf(loggers.Info, "The use of Amazone Web Service Glacier could generate additional costs.\n")
	loggers.Printf(loggers.Info, "The author(s) of this program cannot be held responsible for these additional costs\n")
	loggers.Printf(loggers.Info, "More information about pricing : https://aws.amazon.com/glacier/pricing/\n")
	loggers.Printf(loggers.Info, "####################################################################################\n")
}

func displayWarnIfNotFreeTier(restorationContext *awsutils.RestorationContext) {
	strategy := awsutils.GetDataRetrievalStrategy(restorationContext)
	if strategy != "FreeTier" {
		loggers.Printf(loggers.Warning, "##################################################################################################################\n")
		loggers.Printf(loggers.Warning, "Your data retrieval strategy is \"%v\", the next operations could generate additional costs !!!\n", strategy)
		loggers.Printf(loggers.Warning, "Select strategy \"FreeTier\" to avoid these costs :\n")
		loggers.Printf(loggers.Warning, "http://docs.aws.amazon.com/amazonglacier/latest/dev/data-retrieval-policy.html#data-retrieval-policy-using-console\n")
		loggers.Printf(loggers.Warning, "##################################################################################################################\n")
	}
}





