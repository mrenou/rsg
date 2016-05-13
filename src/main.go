package main

import (
	"./vault"
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
)

type Options struct {
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

	restorationContext := awsutils.CreateRestorationContext(sessionValue, accountId, region, vaultName)

	//listJobs(restorationContext.GlacierClient, accountId, restorationContext.MappingVault)

	vault.DownloadMappingArchive(restorationContext)
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
	flag.StringVarP(&options.region, "region", "r", "", "region of the vault to restore")
	flag.StringVarP(&options.vault, "vault", "v", "", "vault to restore")
	flag.Parse()
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





