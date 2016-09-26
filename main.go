package main

import (
	"rsg/core"
	"rsg/loggers"
	"rsg/awsutils"
	"rsg/utils"
	"rsg/options"
)

func main() {
	loggers.InitDefaultLog()
	options := options.ParseOptions()
	core.DisplayInfoAboutCosts(options)
	session := awsutils.BuildSession(options.AwsId, options.AwsSecret)
	accountId, err := awsutils.GetAccountId(session)
	utils.ExitIfError(err)
	region, vaultName := core.SelectRegionVault(accountId, session, options.Region, options.Vault)
	restorationContext := awsutils.CreateRestorationContext(session, accountId, region, vaultName, options)
	if options.ListJobs {
		core.ListJobs(restorationContext)

	} else {
		core.DownloadMappingArchive(restorationContext)
		core.QueryFiltersIfNecessary(restorationContext, options)
		if options.List {
			core.ListArchives(restorationContext)
		} else {
			err = core.CheckDestinationDirectory(restorationContext)
			utils.ExitIfError(err)
			core.DownloadArchives(restorationContext)
		}
	}

}


