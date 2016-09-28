package core

import (
	"rsg/loggers"
)

func ListArchives(restorationContext *RestorationContext) {
	db := InitDb(restorationContext.GetMappingFilePath())
	defer db.Close()

	archiveRows := GetFiles(db, restorationContext.Options.Filters)
	defer archiveRows.Close()

	for archiveRows.Next() {
		var basePath string
		archiveRows.Scan(&basePath)
		loggers.Printfln(loggers.Info, "%v", basePath)
	}
}
