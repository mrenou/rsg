package utils

import "os"
import (
	"../loggers"
)

const S_1MB = 1024 * 1024
const S_1GB = 1024 * S_1MB

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

