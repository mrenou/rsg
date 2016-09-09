package vault

import (
	"os"
	"errors"
	"fmt"
	"rsg/inputs"
	"rsg/awsutils"
	"rsg/loggers"
)

func CheckDestinationDirectory(restorationContext *awsutils.RestorationContext) error {
	if restorationContext.DestinationDirPath == "" {
		restorationContext.DestinationDirPath = inputs.QueryString("what is the destination directory path ?")
	}
	loggers.Printf(loggers.Info, "destination directory path is %v\n", restorationContext.DestinationDirPath)
	if stat, err := os.Stat(restorationContext.DestinationDirPath); !os.IsNotExist(err) {
		if !stat.IsDir() {
			return errors.New(fmt.Sprintf("destination directory is a file: %s", restorationContext.DestinationDirPath))
		}
		if !queryAndUpdateKeepFiles(restorationContext) {
			os.RemoveAll(restorationContext.DestinationDirPath)
		}
	}
	if err := os.MkdirAll(restorationContext.DestinationDirPath, 0700); err != nil {
		return errors.New(fmt.Sprintf("cannot create destination directory: %s", restorationContext.DestinationDirPath))
	}
	return nil
}

func queryAndUpdateKeepFiles(restorationContext *awsutils.RestorationContext) bool {
	for restorationContext.Options.KeepFiles == nil {
		if !inputs.QueryYesOrNo("destination directory already exists, do you want to keep existing files ?", true) {
			if inputs.QueryYesOrNo("are you sure, all existing files restored will be deleted ?", false) {
				tmp := false
				restorationContext.Options.KeepFiles = &tmp
			}
		} else {
			tmp := true
			restorationContext.Options.KeepFiles = &tmp
		}
	}
	return *restorationContext.Options.KeepFiles
}
