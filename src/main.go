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
	"errors"
)

type Options struct {
	dest   string
	region string
	vault  string
}

func main() {
	loggers.InitDefaultLog()
	sessionValue := session.New()
	accountId, err := getAccountId(sessionValue)
	utils.ExitIfError(err)
	options := parseOptions()
	region, vaultName := vault.SelectRegionVault(accountId, sessionValue, options.region, options.vault)
	loggers.DebugPrintf("region and vault used for restauration : %s:%s", region, vaultName)


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
	if err = os.MkdirAll(options.dest, 0700); err != nil {
		utils.ExitIfError(errors.New(fmt.Sprintf("cannot create destination directory: %s", options.dest)))
	}


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
	flag.Parse()

	if (flag.NArg() != 1) {
		fmt.Fprintf(os.Stderr, "no destination given\n")
		os.Exit(2)
	}
	options.dest = flag.Arg(0)

	loggers.DebugPrintf("options dest=%v \n", options.dest)
	loggers.DebugPrintf("options region=%v \n", options.region)
	loggers.DebugPrintf("options vault=%v \n", options.vault)
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




