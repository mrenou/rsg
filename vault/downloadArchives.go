package vault

import (
	"database/sql"
	"os"
	"rsg/utils"
	"rsg/awsutils"
	"rsg/loggers"
	"container/list"
	"path/filepath"
	"errors"
	"time"
	"code.cloudfoundry.org/bytefmt"
	"strings"
	"rsg/inputs"
	"rsg/speedtest"
)

type archiveRetrieve struct {
	archiveId               string
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
	jobId                string
	archiveId            string
	retrievedSize        uint64
	archiveSize          uint64
	nextByteIndexToWrite uint64
}

type ArchiveRetrieveResult int

const (
	SUCCESS ArchiveRetrieveResult = iota
	SKIPPED
	RETRY
)

// + 10 is safety margin
const archiveRetrieveStructSize = 92 + 138 + 8 + 8 + 8 + 10
const _4hoursInSeconds = 60 * 60 * 4
const _5minInSeconds = 60 * 5

type DownloadContext struct {
	restorationContext             *awsutils.RestorationContext
	bytesBySecond                  uint64
	downloadSpeedAutoUpdate        bool
	nbBytesToDownload              uint64
	nbBytesDownloaded              uint64
	maxArchivesRetrievingSize      uint64
	archivesRetrievingSize         uint64
	archivePartRetrieveListMaxSize int
	archivePartRetrieveList        *list.List
	hasArchiveRows                 bool
	db                             *sql.DB
	archiveRows                    *sql.Rows
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
	downloadContext.downloadSpeedAutoUpdate = true
	downloadContext.archivePartRetrieveListMaxSize = utils.S_1GB / archiveRetrieveStructSize
	if downloadContext.bytesBySecond == 0 {
		downloadContext.bytesBySecond = detectOrSelectDownloadSpeed(restorationContext)
	}
	downloadContext.maxArchivesRetrievingSize = downloadContext.bytesBySecond * uint64(_4hoursInSeconds)
	downloadContext.downloadArchives()
}

func detectOrSelectDownloadSpeed(restorationContext *awsutils.RestorationContext) uint64 {
	downloadSpeed, err := speedtest.SpeedTest()
	if err != nil {
		loggers.Printfln(loggers.Error, "Cannot test download speed : %v", err)
		for downloadSpeed == 0 || err != nil {
			downloadSpeed, err = bytefmt.ToBytes(inputs.QueryString("Select your download speed by second (ex 10K, 256K, 1M, 10M):"))
			if err != nil {
				loggers.Printfln(loggers.Error, "%v", err)
			}
		}
	}
	loggers.Printfln(loggers.OptionalInfo, "Download speed used : %v", bytefmt.ByteSize(downloadSpeed))
	return downloadSpeed
}

func (downloadContext *DownloadContext) downloadArchives() {
	DisplayWarnIfNotFreeTier(downloadContext.restorationContext)
	if (downloadContext.maxArchivesRetrievingSize < utils.S_1MB) {
		utils.ExitIfError(errors.New("Max archives retrieving size cannot be less than 1MB"))
	}

	db := InitDb(downloadContext.restorationContext.GetMappingFilePath())
	downloadContext.db = db
	defer db.Close()

	archiveRows := GetArchives(db, downloadContext.restorationContext.Options.Filters)
	downloadContext.archiveRows = archiveRows
	defer archiveRows.Close()

	downloadContext.nbBytesToDownload = GetTotalSize(db, downloadContext.restorationContext.Options.Filters)
	loggers.Printfln(loggers.OptionalInfo, "%v to restore", bytefmt.ByteSize(downloadContext.nbBytesToDownload))

	downloadContext.archivePartRetrieveList = list.New()
	downloadContext.archivesRetrievingSize = 0
	downloadContext.hasArchiveRows = true

	for !downloadContext.allFilesHasBeenProcessed() {
		downloadContext.startArchiveRetrievingJobs()
		downloadContext.downloadArchivesPartWhenReady()
	}
}

func (downloadContext *DownloadContext) allFilesHasBeenProcessed() bool {
	return !downloadContext.hasArchiveRows &&
		downloadContext.archivePartRetrieveList.Len() == 0 &&
		downloadContext.uncompletedRetrieve == nil &&
		downloadContext.uncompletedDownload == nil
}

func (downloadContext *DownloadContext) startArchiveRetrievingJobs() {
	for downloadContext.archivesRetrievingSize < downloadContext.maxArchivesRetrievingSize &&
		downloadContext.archivePartRetrieveList.Len() < downloadContext.archivePartRetrieveListMaxSize &&
		(downloadContext.hasArchiveRows || downloadContext.uncompletedRetrieve != nil) {
		downloadContext.displayStatus("start retrieve jobs")
		if downloadContext.uncompletedRetrieve == nil {
			downloadContext.uncompletedRetrieve = downloadContext.findNextArchiveToRetrieve()
		}
		if downloadContext.uncompletedRetrieve != nil {
			if startStatus := downloadContext.startArchivePartRetrieveJob(downloadContext.uncompletedRetrieve); startStatus == RETRY {
				break
			}
		}
	}
}

func (downloadContext *DownloadContext) findNextArchiveToRetrieve() *archiveRetrieve {
	var archiveToRetrieve *archiveRetrieve;
	for archiveToRetrieve == nil && downloadContext.hasArchiveRows {
		downloadContext.hasArchiveRows = downloadContext.archiveRows.Next()
		if downloadContext.hasArchiveRows {
			var archiveId string
			var fileSize uint64
			err := downloadContext.archiveRows.Scan(&archiveId, &fileSize)
			utils.ExitIfError(err)

			if !downloadContext.checkAllFilesOfArchiveExists(archiveId) {
				if stat, err := os.Stat(downloadContext.restorationContext.DestinationDirPath + "/" + archiveId); !os.IsNotExist(err) {
					loggers.Printfln(loggers.Verbose, "Local archive found: %v", archiveId)
					if !downloadContext.handleArchiveFileDownloadCompletion(archiveId, fileSize) {
						archiveToRetrieve = &archiveRetrieve{archiveId, fileSize, uint64(stat.Size()) - (uint64(stat.Size()) % utils.S_1MB)}
					}
				} else if fileSize == 0 {
					downloadContext.createFilesForEmptyArchive(archiveId)
				} else {
					archiveToRetrieve = &archiveRetrieve{archiveId, fileSize, 0}
				}
			}
		}
	}
	return archiveToRetrieve
}

func (downloadContext *DownloadContext) checkAllFilesOfArchiveExists(archiveId string) bool {
	pathRows := GetPaths(downloadContext.db, archiveId)
	for pathRows.Next() {
		var path string
		pathRows.Scan(&path)

		if utils.Exists(downloadContext.restorationContext.DestinationDirPath + "/" + path) {
			loggers.Printfln(loggers.Verbose, "Skip existing file %s", downloadContext.restorationContext.DestinationDirPath + "/" + path)
		} else {
			loggers.Printfln(loggers.Verbose, "File not found: %v/%v", downloadContext.restorationContext.DestinationDirPath, path)
			return false

		}
	}
	pathRows.Close()
	return true
}

func (downloadContext *DownloadContext) createFilesForEmptyArchive(archiveId string) {
	pathRows := GetPaths(downloadContext.db, archiveId)
	for pathRows.Next() {
		var path string
		pathRows.Scan(&path)
		if !utils.Exists(downloadContext.restorationContext.DestinationDirPath + "/"+ path) {
			err := os.MkdirAll(filepath.Dir(downloadContext.restorationContext.DestinationDirPath + "/"+ path), 0700)
			utils.ExitIfError(err)
			file, err := os.Create(downloadContext.restorationContext.DestinationDirPath + "/"+ path)
			utils.ExitIfError(err)
			err = file.Close()
			utils.ExitIfError(err)
		}
	}
	pathRows.Close()
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

func (downloadContext *DownloadContext) startArchivePartRetrieveJob(archiveToRetrieve *archiveRetrieve) ArchiveRetrieveResult {
	sizeToRetrieve, isEndOfFile := downloadContext.computeSizeToRetrieve(downloadContext.uncompletedRetrieve)
	if (isEndOfFile || sizeToRetrieve / utils.S_1MB > 0) {
		startStatus, jobId, sizeRetrieved := downloadContext.retryArchivePartRetrieveJob(archiveToRetrieve, sizeToRetrieve)
		if startStatus == SUCCESS {
			loggers.Printfln(loggers.Verbose, "Job has started for archive id %s to retrieve %v from %v byte index",
				archiveToRetrieve.archiveId,
				bytefmt.ByteSize(sizeRetrieved),
				archiveToRetrieve.nextByteIndexToRetrieve)
			archivePartRetrieve := &archivePartRetrieve{jobId, archiveToRetrieve.archiveId, sizeRetrieved, archiveToRetrieve.size, archiveToRetrieve.nextByteIndexToRetrieve}
			archiveToRetrieve.nextByteIndexToRetrieve += sizeRetrieved
			downloadContext.archivesRetrievingSize += sizeRetrieved
			downloadContext.archivePartRetrieveList.PushFront(archivePartRetrieve)
			downloadContext.handleArchiveRetrieveCompletion(archiveToRetrieve)
		}
		return startStatus
	}
	return RETRY
}

func (downloadContext *DownloadContext) retryArchivePartRetrieveJob(archiveToRetrieve *archiveRetrieve, sizeToRetrieve uint64) (ArchiveRetrieveResult, string, uint64) {
	var jobId string
	var sizeRetrieved uint64
	var err error
	for {
		jobId, sizeRetrieved, err = awsutils.StartRetrievePartialArchiveJob(downloadContext.restorationContext,
			downloadContext.restorationContext.Vault,
			awsutils.Archive{archiveToRetrieve.archiveId, archiveToRetrieve.size},
			archiveToRetrieve.nextByteIndexToRetrieve,
			sizeToRetrieve)
		if err == nil {
			return SUCCESS, jobId, sizeRetrieved
		}
		if strings.Contains(err.Error(), "PolicyEnforcedException") {
			if (downloadContext.uncompletedDownload != nil) {
				return RETRY, "", 0
			}
			downloadContext.displayStatus("rate limit reached, waiting")
			time.Sleep(5 * time.Minute)

		} else if strings.Contains(err.Error(), "ResourceNotFoundException") {
			loggers.Printfln(loggers.Warning, "Archive not found %s, skipped...", archiveToRetrieve.archiveId)
			downloadContext.uncompletedRetrieve = nil
			return SKIPPED, "", 0
		} else {
			utils.ExitIfError(err)
		}
	}
}

func (downloadContext *DownloadContext) handleArchiveRetrieveCompletion(archiveToRetrieve *archiveRetrieve) {
	if archiveToRetrieve.retrieveIsComplete() {
		loggers.Printfln(loggers.Verbose, "Archive id %s has been completed retrieved", archiveToRetrieve.archiveId)
		downloadContext.uncompletedRetrieve = nil
	}
}

func (downloadContext *DownloadContext) downloadArchivesPartWhenReady() {
	maxArchivesDownloadingSize := downloadContext.bytesBySecond * uint64(_5minInSeconds)
	var archivesDownloadingSize uint64 = 0
	totalDuration := time.Duration(0)

	for archivesDownloadingSize < maxArchivesDownloadingSize && (downloadContext.archivePartRetrieveList.Len() > 0 || downloadContext.uncompletedDownload != nil) {
		if (downloadContext.uncompletedDownload == nil && downloadContext.archivePartRetrieveList.Len() > 0) {
			downloadContext.displayStatus("wait archive retrieve job")
			downloadContext.uncompletedDownload = downloadContext.waitNextArchivePartIsRetrieved()
		}
		downloadContext.displayStatus("downloading")
		archivesDownloadingSizeLeft := maxArchivesDownloadingSize - archivesDownloadingSize
		sizeDownloaded, duration := downloadArchivePart(downloadContext.restorationContext, downloadContext.uncompletedDownload, downloadContext.nextByteIndexToDownload, archivesDownloadingSizeLeft)
		totalDuration += duration
		archivesDownloadingSize += sizeDownloaded
		downloadContext.nbBytesDownloaded += sizeDownloaded
		downloadContext.nextByteIndexToDownload += sizeDownloaded
		downloadContext.archivesRetrievingSize -= sizeDownloaded

		downloadContext.handleArchivePartDownloadCompletion(downloadContext.restorationContext)
	}
	downloadContext.updateDownloadSpeed(archivesDownloadingSize, totalDuration)
}

func (downloadContext *DownloadContext) displayStatus(phase string) {
	restored := uint64(0)
	if downloadContext.nbBytesToDownload != 0 {
		restored = downloadContext.nbBytesDownloaded * 100 / downloadContext.nbBytesToDownload
	}
	if (loggers.VerboseFlag) {
		loggers.Printfln(loggers.Info, "%-30s %02v%% restored", "(" + phase + ")", restored)
	} else {
		loggers.Printf(loggers.Info, "\r%-30s %02v%% restored", "(" + phase + ")", restored)
	}
}

func (downloadContext *DownloadContext) updateDownloadSpeed(downloadedSize uint64, duration time.Duration) {
	if (downloadContext.downloadSpeedAutoUpdate) {
		downloadContext.bytesBySecond = uint64(float64(downloadedSize) / duration.Seconds())
		if (downloadContext.bytesBySecond == 0) {
			downloadContext.bytesBySecond = 1
		}
		loggers.Printfln(loggers.Verbose, "New download speed: %v/s", bytefmt.ByteSize(downloadContext.bytesBySecond))
	}
}

func (downloadContext *DownloadContext) handleArchivePartDownloadCompletion(restorationContext *awsutils.RestorationContext) {
	if (downloadContext.nextByteIndexToDownload >= downloadContext.uncompletedDownload.retrievedSize) {
		downloadContext.handleArchiveFileDownloadCompletion(downloadContext.uncompletedDownload.archiveId, downloadContext.uncompletedDownload.archiveSize)
		downloadContext.uncompletedDownload = nil
		downloadContext.nextByteIndexToDownload = 0
	}
}

func (downloadContext *DownloadContext) handleArchiveFileDownloadCompletion(archiveId string, size uint64) bool {
	destinationDirPath := downloadContext.restorationContext.DestinationDirPath
	file, err := os.Open(destinationDirPath + "/" + archiveId)
	utils.ExitIfError(err)
	stat, err := file.Stat()
	utils.ExitIfError(err)
	if uint64(stat.Size()) >= size {
		loggers.Printfln(loggers.Verbose, "Archive %v downloaded", archiveId)

		pathRows := GetPaths(downloadContext.db, archiveId)
		defer pathRows.Close()
		var previousPath string
		if pathRows.Next() {
			pathRows.Scan(&previousPath)
		}
		for pathRows.Next() {
			var path string
			pathRows.Scan(&path)
			if !utils.Exists(destinationDirPath + "/" + previousPath) {
				err := os.MkdirAll(filepath.Dir(destinationDirPath + "/" + previousPath), 0700)
				utils.ExitIfError(err)
				utils.CopyFile(destinationDirPath + "/" + previousPath, destinationDirPath + "/" + archiveId)
				loggers.Printfln(loggers.Verbose, "File %v restored (copy from %v)", destinationDirPath + "/" + previousPath, archiveId)
			}
			previousPath = path;
		}
		if previousPath != "" && !utils.Exists(destinationDirPath + "/" + previousPath) {
			err := os.MkdirAll(filepath.Dir(destinationDirPath + "/" + previousPath), 0700)
			utils.ExitIfError(err)
			os.Rename(destinationDirPath + "/" + archiveId, destinationDirPath + "/" + previousPath)
			loggers.Printfln(loggers.Verbose, "File %v restored (rename from %v)", destinationDirPath + "/" + previousPath, archiveId)
		}
		return true
	}
	return false
}

func (downloadContext *DownloadContext) waitNextArchivePartIsRetrieved() *archivePartRetrieve {
	element := downloadContext.archivePartRetrieveList.Back()
	archivePartRetrieve := element.Value.(*archivePartRetrieve)
	awsutils.WaitJobIsCompleted(downloadContext.restorationContext, downloadContext.restorationContext.Vault, archivePartRetrieve.jobId)
	downloadContext.archivePartRetrieveList.Remove(element)
	return archivePartRetrieve
}

func downloadArchivePart(restorationContext *awsutils.RestorationContext, archivePartRetrieve *archivePartRetrieve, fromByteIndex, nbBytesCanDownload uint64) (uint64, time.Duration) {
	sizeToDownload := archivePartRetrieve.retrievedSize - fromByteIndex
	if (sizeToDownload > nbBytesCanDownload) {
		sizeToDownload = nbBytesCanDownload
	}
	start := time.Now()
	sizeDownloaded := awsutils.DownloadPartialArchiveTo(restorationContext,
		restorationContext.Vault,
		archivePartRetrieve.jobId,
		restorationContext.DestinationDirPath + "/" + archivePartRetrieve.archiveId,
		fromByteIndex,
		sizeToDownload,
		archivePartRetrieve.nextByteIndexToWrite)
	archivePartRetrieve.nextByteIndexToWrite += sizeDownloaded
	return sizeDownloaded, time.Since(start)
}
