package awsutils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/service/glacier/glacieriface"
	"github.com/aws/aws-sdk-go/aws/session"
	"rsg/utils"
)

// Loaded in package variable because don't change after vault selection
var AccountId string
var Session *session.Session

func LoadAccountSession(awsId, awsSecret string)  {
	var err error
	Session = BuildSession(awsId, awsSecret)
	AccountId, err = GetAccountId(Session)
	utils.ExitIfError(err)
}

func GetVaults(glacierClient glacieriface.GlacierAPI, marker *string) (*glacier.ListVaultsOutput, error) {
	params := &glacier.ListVaultsInput{
		AccountId: aws.String(AccountId),
		Marker:    marker,
	}
	return glacierClient.ListVaults(params)
}