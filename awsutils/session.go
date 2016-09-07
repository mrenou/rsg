package awsutils

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws"
)

func BuildSession(awsId, awsSecret string) *session.Session {
	var sessionValue *session.Session
	if (awsId != "" && awsSecret != "") {
		credentialsValue := credentials.NewStaticCredentials(awsId, awsSecret, "")
		sessionValue = session.New(&aws.Config{Credentials: credentialsValue})
	} else {
		sessionValue = session.New()
	}
	return sessionValue;
}
