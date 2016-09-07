package main

import (
	"rsg/vault"
	"github.com/aws/aws-sdk-go/aws/session"
	flag "github.com/spf13/pflag"
	"strings"
	"github.com/aws/aws-sdk-go/service/iam"
	"rsg/loggers"
	"rsg/awsutils"
	"rsg/utils"
	"fmt"
	"os"
	_ "github.com/mattn/go-sqlite3"
)

type Options struct {
	dest    string
	region  string
	vault   string
	debug   bool
	filters []string
}

func main() {
	loggers.InitDefaultLog()
	sessionValue := session.New()
	accountId, err := getAccountId(sessionValue)
	utils.ExitIfError(err)

	options := parseOptions()
	region, vaultName := vault.SelectRegionVault(accountId, sessionValue, options.region, options.vault)
	restorationContext := awsutils.CreateRestorationContext(sessionValue, accountId, region, vaultName, options.dest)

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

func getAccountId(sessionValue *session.Session) (string, error) {
	svc := iam.New(sessionValue)
	params := &iam.GetUserInput{}
	resp, err := svc.GetUser(params)
	if err != nil {
		return "", err
	}
	return strings.Split(*resp.User.Arn, ":")[4], nil;
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





