package vault

import (
	"database/sql"
	"strings"
	"rsg/loggers"
	"rsg/utils"
)

func InitDb(file string) *sql.DB {
	db, err := sql.Open("sqlite3", file)
	utils.ExitIfError(err)
	return db
}

func GetFiles(db *sql.DB, filters []string) *sql.Rows {
	where := buildWhereFromFilters(filters)
	sqlQuery := "SELECT basePath FROM file_info_tb " + where + " ORDER BY basePath"
	rows, err := db.Query(sqlQuery)
	utils.ExitIfError(err)
	return rows
}

func GetArchives(db *sql.DB, filters []string) *sql.Rows {
	where := buildWhereFromFilters(filters)
	sqlQuery := "SELECT DISTINCT archiveId, fileSize FROM file_info_tb " + where + " ORDER BY key"
	loggers.Printf(loggers.Verbose, "Query mapping file for archives: %v\n", sqlQuery)
	rows, err := db.Query(sqlQuery)
	utils.ExitIfError(err)
	return rows
}

func GetPaths(db *sql.DB, archiveId string) *sql.Rows {
	stmt, err := db.Prepare("SELECT DISTINCT basePath FROM file_info_tb WHERE archiveId = ?")
	utils.ExitIfError(err)
	defer stmt.Close()
	rows, err := stmt.Query(archiveId)
	utils.ExitIfError(err)
	return rows
}

func GetTotalSize(db *sql.DB, filters []string) uint64 {
	where := buildWhereFromFilters(filters)
	//row := db.QueryRow("SELECT sum(fileSize) FROM file_info_tb " + where + "GROUP BY archiveId")
	row := db.QueryRow("SELECT sum(t.fileSize) FROM (SELECT fileSize FROM file_info_tb " + where + " GROUP BY archiveId) t")
	var totalSize uint64
	err := row.Scan(&totalSize)
	utils.ExitIfError(err)
	return totalSize
}

func buildWhereFromFilters(filters []string) string {
	where := ""
	if len(filters) > 0 {
		where = "WHERE "
		for i, filter := range filters {
			filter = strings.Replace(filter, "*", "%", -1)
			filter = strings.Replace(filter, "?", "_", -1)
			if i > 0 {
				where += " OR "
			}
			where += "basepath LIKE '" + filter + "'"
		}
	}
	return where
}
