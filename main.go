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
	options := options.ParseOptions()
	vault.DisplayInfoAboutCosts(options)
	session := awsutils.BuildSession(options.AwsId, options.AwsSecret)
	accountId, err := awsutils.GetAccountId(session)
	utils.ExitIfError(err)
	region, vaultName := vault.SelectRegionVault(accountId, session, options.Region, options.Vault)
	restorationContext := awsutils.CreateRestorationContext(session, accountId, region, vaultName, options)
	vault.DownloadMappingArchive(restorationContext)
	if options.List {
		vault.ListArchives(restorationContext)
	} else {
		err = vault.CheckDestinationDirectory(restorationContext)
		utils.ExitIfError(err)
		vault.DownloadArchives(restorationContext)
	}
}


