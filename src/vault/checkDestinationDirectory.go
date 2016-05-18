package vault

import (
	"os"
	"errors"
	"fmt"
	"../inputs"
	"../awsutils"
)

func CheckDestinationDirectory(restorationContext *awsutils.RestorationContext) error {
	destinationDirectoryPath := restorationContext.DestinationDirPath
	if stat, err := os.Stat(destinationDirectoryPath); !os.IsNotExist(err) {
		if !stat.IsDir() {
			return errors.New(fmt.Sprintf("destination directory is a file: %s", destinationDirectoryPath))
		}
		if !inputs.QueryYesOrNo("destination directory already exists, do you want to keep existing files ?", true) {
			if inputs.QueryYesOrNo("are you sure, all existing files restored will be deleted ?", false) {
				os.RemoveAll(destinationDirectoryPath)
			}
		}
	}
	if err := os.Mkdir(destinationDirectoryPath, 0700); err != nil {
		return errors.New(fmt.Sprintf("cannot create destination directory: %s", destinationDirectoryPath))
	}
	return nil
}
