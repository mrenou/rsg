package utils

import "os"
import (
	"rsg/loggers"
	"io"
	"time"
	"strings"
	"errors"
)

const S_1MB = 1024 * 1024
const S_1GB = 1024 * S_1MB

func ExitIfError(err error) {
	if (err != nil) {
		loggers.Printf(loggers.Error, "%v\n", translateAwsErrors(err))
		os.Exit(1)
	}
}

func translateAwsErrors(err error) error {
	if strings.Contains(err.Error(),"NoCredentialProviders") {
		return errors.New("No credentials found, check your ~/.aws/credentials (http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html#cli-config-files) or give them as arguments (aws-id and aws-secret)")
	}
	if strings.Contains(err.Error(), "SignatureDoesNotMatch") {
		return errors.New("Signature does not match, check your credentials")
	}
	return err
}

func Contains(values []string, toFind string) bool {
	for _, value := range values {
		if value == toFind {
			return true
		}
	}
	return false
}

func CopyFile(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	// no need to check errors on read only file, we already got everything
	// we need from the filesystem, so nothing can go wrong now.
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

func WaitNextHour() {
	actualHour := time.Now().Hour()
	for {
		time.Sleep(time.Duration(60 - time.Now().Minute()) * time.Minute)
		if time.Now().Hour() != actualHour {
			break
		}
	}
}


