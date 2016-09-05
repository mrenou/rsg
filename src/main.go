package main

import (
	"./vault"
	"./inputs"
	"github.com/aws/aws-sdk-go/aws/session"
	flag "github.com/spf13/pflag"
	"strings"
	"github.com/aws/aws-sdk-go/service/iam"
	"./loggers"
	"./awsutils"
	"./utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glacier"
	"fmt"
	"os"
	_ "github.com/mattn/go-sqlite3"
)

type Options struct {
	dest   string
	region string
	vault  string
	debug  bool
}

func main() {
	loggers.InitDefaultLog()
	sessionValue := session.New()
	accountId, err := getAccountId(sessionValue)
	utils.ExitIfError(err)
	options := parseOptions()
	region, vaultName := vault.SelectRegionVault(accountId, sessionValue, options.region, options.vault)
	loggers.Printf(loggers.Debug, "region and vault used for restauration : %s:%s\n", region, vaultName)

	restorationContext := awsutils.CreateRestorationContext(sessionValue, accountId, region, vaultName, options.dest)

	//listJobs(restorationContext.GlacierClient, accountId, restorationContext.MappingVault)

	vault.DownloadMappingArchive(restorationContext)
	if _, err := os.Stat(options.dest); os.IsExist(err) {
		if !inputs.QueryYesOrNo("destination directory already exists, do you want to keep existing files ?", true) {
			if !inputs.QueryYesOrNo("are you sure, all existing files restored will be deleted ?", false) {
				os.RemoveAll(options.dest)
			}
		}
	}
	err = vault.CheckDestinationDirectory(restorationContext)
	utils.ExitIfError(err)

	vault.DownloadArchives(restorationContext)
}

func listJobs(glacierClient *glacier.Glacier, accountId, vault string) {
	params := &glacier.ListJobsInput{
		AccountId:  aws.String(accountId), // Required
		VaultName:  aws.String(vault), // Required
	}
	resp, err := glacierClient.ListJobs(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

func getOutputJob(glacierClient *glacier.Glacier, accountId, vault, jobId string) (*[]byte, error) {
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(accountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
		Range:     nil,
	}
	resp, err := glacierClient.GetJobOutput(params)
	if err != nil {
		return nil, err
	}

	fmt.Println(resp)
	//body, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(body))
	return nil, nil
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
	flag.Parse()

	if (flag.NArg() != 1) {
		fmt.Fprintf(os.Stderr, "no destination given\n")
		os.Exit(2)
	}
	options.dest = flag.Arg(0)

	loggers.DebugFlag = options.debug
	loggers.Printf(loggers.Debug, "options dest=%v \n", options.dest)
	loggers.Printf(loggers.Debug, "options region=%v \n", options.region)
	loggers.Printf(loggers.Debug, "options vault=%v \n", options.vault)
	loggers.Printf(loggers.Debug, "options debug=%v \n", options.debug)
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





