package utils

import "os"
import (
	"../loggers"
)

func ExitIfError(err error) {
	if (err != nil) {
		loggers.Error.Print(err)
		os.Exit(1)
	}
}

func Contains(values []string, toFind string) bool {
	for _, value := range values {
		if value == toFind {
			return true
		}
	}
	return false
}

