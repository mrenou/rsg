package main

import (
	"rsg/core"
	"rsg/outputs"
	"rsg/awsutils"
	"rsg/utils"
	"rsg/options"
)

func main() {
	outputs.InitDefaultOutputs()
	options := options.ParseOptions()
	core.DisplayInfoAboutCosts(options)
	awsutils.LoadAccountSession(options.AwsId, options.AwsSecret)
	region, vaultName := core.SelectRegionVault(options.Region, options.Vault)
	restorationContext := core.CreateRestorationContext(region, vaultName, options)

	if options.ListJobs {
		core.ListJobs(restorationContext)

	} else {
		awsutils.LoadJobIdsAtStartup(restorationContext.GlacierClient, restorationContext.MappingVault, restorationContext.Vault)
		core.DownloadMappingArchive(restorationContext)
		core.QueryFiltersIfNecessary(restorationContext, options)
		if options.List {
			core.ListArchives(restorationContext)
		} else {
			err := core.CheckDestinationDirectory(restorationContext)
			utils.ExitIfError(err)
			core.DownloadArchives(restorationContext)
		}
	}

}


