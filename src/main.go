package main

import (
	"./vault"
	"github.com/aws/aws-sdk-go/aws/session"
	flag "github.com/spf13/pflag"
	"strings"
	"github.com/aws/aws-sdk-go/service/iam"
	"./loggers"
	"./utils"
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
	region, vault := vault.SelectRegionVault(accountId, sessionValue, options.region, options.vault)
	loggers.DebugPrintf("region and vault used for restauration : %s:%s", region, vault)
}

func parseOptions() Options {
	options := Options{}
	flag.StringVarP(&options.region, "region", "r", "", "region of the vault to restore")
	flag.StringVarP(&options.region, "vault", "v", "", "vault to restore")
	flag.Parse()
	loggers.DebugPrintf("options %v \n", options)
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





