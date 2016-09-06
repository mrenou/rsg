package utils

import "os"
import (
	"rsg/loggers"
	"io"
)

const S_1MB = 1024 * 1024
const S_1GB = 1024 * S_1MB

func ExitIfError(err error) {
	if (err != nil) {
		loggers.Print(loggers.Error, err)
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

