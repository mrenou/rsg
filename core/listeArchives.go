package core

import (
	"rsg/outputs"
)

// List paths from the mapping file

func ListArchives(restorationContext *RestorationContext) {
	db := InitDb(restorationContext.GetMappingFilePath())
	defer db.Close()

	archiveRows := GetFiles(db, restorationContext.Options.Filters)
	defer archiveRows.Close()

	for archiveRows.Next() {
		var basePath string
		archiveRows.Scan(&basePath)
		outputs.Printfln(outputs.Info, "%v", basePath)
	}
}
