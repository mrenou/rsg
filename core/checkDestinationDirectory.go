package core

import (
	"os"
	"errors"
	"fmt"
	"rsg/inputs"
	"rsg/outputs"
)

// check if destination directory is defined or ask it
// ask if we need to keep files if already exists or restart new restoration

func CheckDestinationDirectory(restorationContext *RestorationContext) error {
	for {
		if restorationContext.DestinationDirPath == "" {
			restorationContext.DestinationDirPath = inputs.QueryString("What is the destination directory path ?")
		}
		outputs.Printfln(outputs.OptionalInfo, "Destination directory path is %v", restorationContext.DestinationDirPath)
		if stat, err := os.Stat(restorationContext.DestinationDirPath); !os.IsNotExist(err) {
			if !stat.IsDir() {
				return errors.New(fmt.Sprintf("Destination directory is a file: %s", restorationContext.DestinationDirPath))
			}
			if !queryAndUpdateKeepFiles(restorationContext) {
				os.RemoveAll(restorationContext.DestinationDirPath)
			}
		}
		if err := os.MkdirAll(restorationContext.DestinationDirPath, 0700); err != nil {
			outputs.Printfln(outputs.Error, "Cannot create destination directory %s : %v", restorationContext.DestinationDirPath, err)
			restorationContext.DestinationDirPath = ""
		} else {
			return nil
		}
	}
}

func queryAndUpdateKeepFiles(restorationContext *RestorationContext) bool {
	for restorationContext.Options.KeepFiles == nil {
		if !inputs.QueryYesOrNo("Destination directory already exists, do you want to keep existing files ?", true) {
			if inputs.QueryYesOrNo("Are you sure, all existing files restored will be deleted ?", false) {
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
