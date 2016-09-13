package awsutils

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"rsg/loggers"
	"strings"
	"github.com/aws/aws-sdk-go/service/iam"
)

func GetAccountId(sessionValue *session.Session) (string, error) {
	svc := iam.New(sessionValue)
	params := &iam.GetUserInput{}
	loggers.Printfln(loggers.Verbose, "Aws call: svc.GetUser(%+v)", params)
	resp, err := svc.GetUser(params)
	if err != nil {
		return "", err
	}
	return strings.Split(*resp.User.Arn, ":")[4], nil;
}
