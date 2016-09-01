package vault

import (
	"database/sql"
	"os"
	"../utils"
	"../awsutils"
	"../loggers"
	"container/list"
	_ "github.com/mattn/go-sqlite3"
	"path/filepath"
	"errors"
)

type archiveRetrieve struct {
	archiveId               string
	dbKey                   uint64
	size                    uint64
	nextByteIndexToRetrieve uint64
}

func (archiveRetrieve *archiveRetrieve) sizeToRetrieveLeft() uint64 {
	return archiveRetrieve.size - archiveRetrieve.nextByteIndexToRetrieve
}

func (archiveRetrieve *archiveRetrieve) retrieveIsComplete() bool {
	return archiveRetrieve.nextByteIndexToRetrieve >= archiveRetrieve.size
}

type archivePartRetrieve struct {
	jobId         string
	archiveId     string
	dbKey         uint64
	retrievedSize uint64
	size          uint64
}

// + 10 is safety margin
const archiveRetrieveStructSize = 92 + 138 + 8 + 8 + 8 + 8 + 10
const _4hoursInSeconds = 60 * 60 * 4
const _5minInSeconds = 60 * 5

type DownloadContext struct {
	restorationContext             *awsutils.RestorationContext
	bytesBySecond                  uint64
	maxArchivesRetrievingSize      uint64
	archivesRetrievingSize         uint64
	archivePartRetrieveListMaxSize int
	archivePartRetrieveList        *list.List
	hasRows                        bool
	db                             *sql.DB
	rows                           *sql.Rows
	uncompletedRetrieve            *archiveRetrieve
	uncompletedDownload            *archivePartRetrieve
	nextByteIndexToDownload        uint64
}

func (downloadContext *DownloadContext) archivesRetrievingSizeLeft() uint64 {
	return downloadContext.maxArchivesRetrievingSize - downloadContext.archivesRetrievingSize
}

func DownloadArchives(restorationContext *awsutils.RestorationContext) {
	downloadContext := new(DownloadContext)
	downloadContext.restorationContext = restorationContext
	downloadContext.bytesBySecond = uint64(utils.S_1MB)
	downloadContext.archivePartRetrieveListMaxSize = utils.S_1GB / archiveRetrieveStructSize
	downloadContext.maxArchivesRetrievingSize = downloadContext.bytesBySecond * uint64(_4hoursInSeconds)
	downloadContext.downloadArchives()
}

func (downloadContext *DownloadContext) downloadArchives() {
	if (downloadContext.maxArchivesRetrievingSize < utils.S_1MB) {
		utils.ExitIfError(errors.New("max archives retrieving size cannot be less than 1MB"))
	}

	db := downloadContext.loadDb()
	defer db.Close()

	rows := downloadContext.loadRows()
	defer rows.Close()

	downloadContext.archivePartRetrieveList = list.New()
	downloadContext.archivesRetrievingSize = 0
	downloadContext.hasRows = true

	for !downloadContext.allFilesHasBeenProcessed() {
		downloadContext.startArchiveRetrievingJobs()
		downloadContext.downloadArchivesPartWhenReady()
	}
}

func (downloadContext *DownloadContext) allFilesHasBeenProcessed() bool {
	return !downloadContext.hasRows &&
	downloadContext.archivePartRetrieveList.Len() == 0 &&
	downloadContext.uncompletedRetrieve == nil &&
	downloadContext.uncompletedDownload == nil
}

func (downloadContext *DownloadContext) startArchiveRetrievingJobs() {
	for downloadContext.archivesRetrievingSize < downloadContext.maxArchivesRetrievingSize &&
	downloadContext.archivePartRetrieveList.Len() < downloadContext.archivePartRetrieveListMaxSize &&
	(downloadContext.hasRows || downloadContext.uncompletedRetrieve != nil) {
		archiveToRetrieve := downloadContext.uncompletedRetrieve

		if archiveToRetrieve == nil {
			archiveToRetrieve = downloadContext.findNextArchiveToRetrieve()
		}
		if archiveToRetrieve != nil {
			sizeToRetreive, isEndOfFile := downloadContext.computeSizeToRetrieve(archiveToRetrieve)
			if (isEndOfFile || sizeToRetreive / utils.S_1MB > 0) {
				downloadContext.startArchivePartRetrieveJob(archiveToRetrieve, sizeToRetreive)
			} else {
				break
			}
		}
	}
}

func (downloadContext *DownloadContext) findNextArchiveToRetrieve() *archiveRetrieve {
	var archiveToRetrieve *archiveRetrieve;
	for archiveToRetrieve == nil && downloadContext.hasRows {
		downloadContext.hasRows = downloadContext.rows.Next()
		if downloadContext.hasRows {
			var dbKey uint64
			var basePath string
			var archiveId string
			var fileSize uint64
			err := downloadContext.rows.Scan(&dbKey, &basePath, &archiveId, &fileSize)
			utils.ExitIfError(err)
			if _, err := os.Stat(downloadContext.restorationContext.DestinationDirPath + "/" + basePath); os.IsNotExist(err) {
				archiveToRetrieve = &archiveRetrieve{archiveId, dbKey, fileSize, 0}
			} else {
				loggers.DebugPrintf("skip existing file %s\n", downloadContext.restorationContext.DestinationDirPath + "/" + basePath)
			}
		}
	}
	return archiveToRetrieve
}

func (downloadContext *DownloadContext) computeSizeToRetrieve(archiveToRetrieve *archiveRetrieve) (uint64, bool) {
	sizeToRetrieve := archiveToRetrieve.sizeToRetrieveLeft()
	archivesRetrievingSizeLeft := downloadContext.archivesRetrievingSizeLeft()
	if (sizeToRetrieve > archivesRetrievingSizeLeft) {
		sizeToRetrieve = archivesRetrievingSizeLeft
		return sizeToRetrieve, false;
	}
	return sizeToRetrieve, true;
}

func (downloadContext *DownloadContext) startArchivePartRetrieveJob(archiveToRetrieve *archiveRetrieve, sizeToRetrieve uint64) {
	jobId, sizeRetrieved := awsutils.StartRetrievePartialArchiveJob(downloadContext.restorationContext,
		downloadContext.restorationContext.Vault,
		awsutils.Archive{archiveToRetrieve.archiveId, archiveToRetrieve.size},
		archiveToRetrieve.nextByteIndexToRetrieve,
		sizeToRetrieve)
	loggers.DebugPrintf("job has started for archive id %s to retrieve %v bytes from %v byte index\n", archiveToRetrieve.archiveId, sizeRetrieved, archiveToRetrieve.nextByteIndexToRetrieve)

	archiveToRetrieve.nextByteIndexToRetrieve += sizeRetrieved

	archivePartRetrieve := &archivePartRetrieve{jobId, archiveToRetrieve.archiveId, archiveToRetrieve.dbKey, sizeRetrieved, archiveToRetrieve.size}
	downloadContext.archivesRetrievingSize += sizeRetrieved
	downloadContext.archivePartRetrieveList.PushFront(archivePartRetrieve)

	downloadContext.handleArchiveRetrieveCompletion(archiveToRetrieve)
}

func (downloadContext *DownloadContext) handleArchiveRetrieveCompletion(archiveToRetrieve *archiveRetrieve) {
	if archiveToRetrieve.retrieveIsComplete() {
		loggers.DebugPrintf("archive id %s has been completed retrieved\n", archiveToRetrieve.archiveId)
		downloadContext.uncompletedRetrieve = nil
	} else {
		downloadContext.uncompletedRetrieve = archiveToRetrieve
	}
}

func (downloadContext *DownloadContext) downloadArchivesPartWhenReady() {
	maxArchivesDownloadingSize := downloadContext.bytesBySecond * uint64(_5minInSeconds)
	var archivesDownloadingSize uint64 = 0

	for archivesDownloadingSize < maxArchivesDownloadingSize && (downloadContext.archivePartRetrieveList.Len() > 0 || downloadContext.uncompletedDownload != nil) {
		if (downloadContext.uncompletedDownload == nil && downloadContext.archivePartRetrieveList.Len() > 0) {
			downloadContext.uncompletedDownload = downloadContext.waitNextArchivePartIsRetrieved()
		}

		archivePath := downloadContext.restorationContext.DestinationDirPath + "/" + getArchiveBasePath(downloadContext.db, downloadContext.uncompletedDownload.dbKey)
		archivesDownloadingSizeLeft := maxArchivesDownloadingSize - archivesDownloadingSize
		sizeDownloaded := downloadArchive(downloadContext.restorationContext, downloadContext.uncompletedDownload, archivePath + ".tmp", downloadContext.nextByteIndexToDownload, archivesDownloadingSizeLeft)
		archivesDownloadingSize += sizeDownloaded
		downloadContext.nextByteIndexToDownload += sizeDownloaded
		downloadContext.archivesRetrievingSize -= sizeDownloaded

		downloadContext.handleArchivePartDownloadCompletion(archivePath)
	}
}

func (downloadContext *DownloadContext) handleArchivePartDownloadCompletion(archiveBasePath string) {
	if (downloadContext.nextByteIndexToDownload >= downloadContext.uncompletedDownload.retrievedSize) {
		downloadContext.handleArchiveFileDownloadCompletion(archiveBasePath)
		downloadContext.uncompletedDownload = nil
		downloadContext.nextByteIndexToDownload = 0
	}
}

func (downloadContext *DownloadContext) handleArchiveFileDownloadCompletion(archiveBasePath string) {
	file, err := os.Open(archiveBasePath + ".tmp")
	utils.ExitIfError(err)
	stat, err := file.Stat()
	utils.ExitIfError(err)
	if uint64(stat.Size()) >= downloadContext.uncompletedDownload.size {
		os.Rename(archiveBasePath + ".tmp", archiveBasePath)
		loggers.DebugPrintf("file %v downloaded\n", downloadContext.restorationContext.DestinationDirPath + "/" + archiveBasePath)
	}
}

func (downloadContext *DownloadContext) waitNextArchivePartIsRetrieved() *archivePartRetrieve {
	element := downloadContext.archivePartRetrieveList.Back()
	archivePartRetrieve := element.Value.(*archivePartRetrieve)
	awsutils.WaitJobIsCompleted(downloadContext.restorationContext, downloadContext.restorationContext.Vault, archivePartRetrieve.jobId)
	downloadContext.archivePartRetrieveList.Remove(element)
	return archivePartRetrieve
}

func downloadArchive(restorationContext *awsutils.RestorationContext, archivePartRetrieve *archivePartRetrieve, archivePath string, fromByteIndex, nbBytesCanDownload uint64) uint64 {
	sizeToDownload := archivePartRetrieve.retrievedSize - fromByteIndex
	if (sizeToDownload > nbBytesCanDownload) {
		sizeToDownload = nbBytesCanDownload
	}
	err := os.MkdirAll(filepath.Dir(archivePath), 0700)
	utils.ExitIfError(err)
	awsutils.DownloadPartialArchiveTo(restorationContext, restorationContext.Vault, archivePartRetrieve.jobId, archivePath, fromByteIndex, sizeToDownload)
	return sizeToDownload
}

func (downloadContext *DownloadContext) loadDb() *sql.DB {
	db, err := sql.Open("sqlite3", downloadContext.restorationContext.GetMappingFilePath())
	utils.ExitIfError(err)
	downloadContext.db = db
	return db
}

func (downloadContext *DownloadContext) loadRows() *sql.Rows {
	rows, err := downloadContext.db.Query("SELECT key, basePath, archiveId, fileSize FROM file_info_tb ORDER BY key")
	utils.ExitIfError(err)
	downloadContext.rows = rows
	return rows
}

func getArchiveBasePath(db *sql.DB, key uint64) string {
	stmt, err := db.Prepare("SELECT basePath FROM file_info_tb where key = ?")
	utils.ExitIfError(err)
	defer stmt.Close()
	row := stmt.QueryRow(key)
	utils.ExitIfError(err)
	var basePath string;
	err = row.Scan(&basePath)
	utils.ExitIfError(err)
	return basePath
}
