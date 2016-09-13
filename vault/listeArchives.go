package vault

import (
	"rsg/awsutils"
	"rsg/loggers"
)

func ListArchives(restorationContext *awsutils.RestorationContext) {
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
