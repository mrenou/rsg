package main

import (
	"rsg/vault"
	flag "github.com/spf13/pflag"
	"rsg/loggers"
	"rsg/awsutils"
	"rsg/utils"
	"fmt"
	"os"
	_ "github.com/mattn/go-sqlite3"
)

type Options struct {
	awsId     string
	awsSecret string
	debug     bool
	dest      string
	filters   []string
	region    string
	vault     string
}

func main() {
	loggers.InitDefaultLog()
	displayInfoAboutCosts()
	options := parseOptions()
	session := awsutils.BuildSession(options.awsId, options.awsSecret)
	accountId, err := awsutils.GetAccountId(session)
	utils.ExitIfError(err)
	region, vaultName := vault.SelectRegionVault(accountId, session, options.region, options.vault)
	restorationContext := awsutils.CreateRestorationContext(session, accountId, region, vaultName, options.dest)
	displayWarnIfNotFreeTier(restorationContext)
	vault.DownloadMappingArchive(restorationContext)
	err = vault.CheckDestinationDirectory(restorationContext)
	utils.ExitIfError(err)
	vault.DownloadArchives(restorationContext)
}

func parseOptions() Options {
	options := Options{}
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n%s [OPTIONS] DEST\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVarP(&options.region, "region", "r", "", "region of the vault to restore")
	flag.StringVarP(&options.vault, "vault", "v", "", "vault to restore")
	flag.BoolVarP(&options.debug, "debug", "x", false, "display debug info")
	flag.StringSliceVarP(&options.filters, "filter", "f", []string{}, "filter files to restore (globals * and ?")
	flag.StringVar(&options.awsId, "aws-id", "", "id of aws credentials")
	flag.StringVar(&options.awsSecret, "aws-secret", "", "secret of aws credentials")
	flag.Parse()

	if (flag.NArg() != 1) {
		fmt.Fprintf(os.Stderr, "no destination given\n")
		flag.Usage()
		os.Exit(2)
	}
	options.dest = flag.Arg(0)

	loggers.DebugFlag = options.debug
	loggers.Printf(loggers.Debug, "options parsed: %+v \n", options)
	return options
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





