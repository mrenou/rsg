package core

import (
	"rsg/loggers"
	"rsg/utils"
	"rsg/inputs"
	"rsg/awsutils"
	"strings"
	"os"
	"fmt"
	"time"
	"code.cloudfoundry.org/bytefmt"
)

func DownloadMappingArchive(restorationContext *awsutils.RestorationContext) {
	if stat, err := os.Stat(restorationContext.GetMappingFilePath()); os.IsNotExist(err) {
		downloadMappingArchive(restorationContext)
	} else if queryAndUpdateRefreshMappingFile(restorationContext, stat.ModTime().Format("Mon Jan _2 15:04:05 2006")) {
			os.Remove(restorationContext.GetMappingFilePath())
			downloadMappingArchive(restorationContext)
	}
}

func queryAndUpdateRefreshMappingFile(restorationContext *awsutils.RestorationContext, modTime string) bool {
	if restorationContext.Options.RefreshMappingFile == nil {
		answer := inputs.QueryYesOrNo(fmt.Sprintf("Local mapping archive already exists with last modification date %v, retrieve a new mapping file ?", modTime), false)
		restorationContext.Options.RefreshMappingFile = &answer
	}
	return *restorationContext.Options.RefreshMappingFile
}

func downloadMappingArchive(restorationContext *awsutils.RestorationContext) {
	jobId, jobCompleted := checkRetrieveMappingOrStartNewJob(restorationContext, getMappingArchive(restorationContext))
	if !jobCompleted {
		awsutils.WaitJobIsCompleted(restorationContext, restorationContext.MappingVault, jobId)
		loggers.Printfln(loggers.OptionalInfo, "Job has finished: %s", jobId)
	}
	start := time.Now()
	sizeDownloaded := awsutils.DownloadArchiveTo(restorationContext, restorationContext.MappingVault, jobId, restorationContext.GetMappingFilePath())
	restorationContext.BytesBySecond = uint64(float64(sizeDownloaded) / time.Since(start).Seconds())
	loggers.Printfln(loggers.Verbose, "New download speed: %v/s", bytefmt.ByteSize(restorationContext.BytesBySecond))
	restorationContext.RegionVaultCache.MappingArchive = nil
	restorationContext.RegionVaultCache.MappingVaultRetrieveJobId = ""
	restorationContext.WriteRegionVaultCache()
	loggers.Println(loggers.OptionalInfo, "Mapping archive has been downloaded")
}

func checkRetrieveMappingOrStartNewJob(restorationContext *awsutils.RestorationContext, archive awsutils.Archive) (string, bool) {
	jobCompleted := false
	jobId := restorationContext.RegionVaultCache.MappingVaultRetrieveJobId
	var err error;
	if jobId != "" {
		loggers.Printfln(loggers.Verbose, "Retrieve mapping archive job id found : %s", jobId)
		jobCompleted, err = awsutils.JobIsCompleted(restorationContext, restorationContext.MappingVault, restorationContext.RegionVaultCache.MappingVaultRetrieveJobId)
		if jobCompleted == false {
			if err == nil {
				loggers.Printfln(loggers.OptionalInfo, "Job to retrieve mapping archive is in progress (can last up to 4 hours): %s", jobId)
			} else if strings.Contains(err.Error(), "The job ID was not found") {
				loggers.Println(loggers.Warning, "Retrieve mapping archive job cached was not found")
				jobId = startRetrieveMappingArchiveJob(restorationContext, restorationContext.MappingVault, archive)
			} else {
				utils.ExitIfError(err)
			}
		}
	} else {
		jobId = startRetrieveMappingArchiveJob(restorationContext, restorationContext.MappingVault, archive)
	}
	return jobId, jobCompleted
}

func startRetrieveMappingArchiveJob(restorationContext *awsutils.RestorationContext, vault string, archive awsutils.Archive) string {
	DisplayWarnIfNotFreeTier(restorationContext)
	jobId, _, err := awsutils.StartRetrieveArchiveJob(restorationContext, restorationContext.MappingVault, archive)
	utils.ExitIfError(err)
	restorationContext.RegionVaultCache.MappingVaultRetrieveJobId = jobId
	restorationContext.WriteRegionVaultCache()
	loggers.Printfln(loggers.OptionalInfo, "Job to retrieve mapping archive has started (can last up to 4 hours): %s", jobId)
	return jobId
}

func getMappingArchive(restorationContext *awsutils.RestorationContext) awsutils.Archive {
	mappingArchive := restorationContext.RegionVaultCache.MappingArchive
	if mappingArchive == nil {
		jobId, jobCompleted := checkMappingInventoryOrStartNewJob(restorationContext)
		if jobCompleted == false {
			awsutils.WaitJobIsCompleted(restorationContext, restorationContext.MappingVault, jobId)
			loggers.Printfln(loggers.OptionalInfo, "Job has finished: %s", jobId)
		}
		restorationContext.RegionVaultCache.MappingVaultInventoryJobId = ""
		restorationContext.RegionVaultCache.MappingArchive = awsutils.GetMappingArchiveIdFromInventory(restorationContext, jobId)
		restorationContext.WriteRegionVaultCache()
	}
	loggers.Printfln(loggers.Verbose, "Mapping archive id is %s", restorationContext.RegionVaultCache.MappingArchive.ArchiveId)
	return *restorationContext.RegionVaultCache.MappingArchive
}

func checkMappingInventoryOrStartNewJob(restorationContext *awsutils.RestorationContext) (string, bool) {
	jobCompleted := false
	jobId := restorationContext.RegionVaultCache.MappingVaultInventoryJobId
	var err error
	if jobId != "" {
		loggers.Printfln(loggers.Verbose, "Mapping vault inventory job id found : %s", jobId)
		jobCompleted, err = awsutils.JobIsCompleted(restorationContext, restorationContext.MappingVault, jobId)
		if jobCompleted == false {
			if err == nil {
				loggers.Printfln(loggers.OptionalInfo, "Job to find mapping archive id is in progress (can last up to 4 hours): %s", jobId)
			} else if strings.Contains(err.Error(), "The job ID was not found") {
				loggers.Println(loggers.Warning, "Inventory job cahed for mapping vaul was not found")
				jobId = inventoryMappingVault(restorationContext)
			} else {
				utils.ExitIfError(err)
			}
		}
	} else {
		jobId = inventoryMappingVault(restorationContext)
	}
	return jobId, jobCompleted
}

func inventoryMappingVault(restorationContext *awsutils.RestorationContext) string {
	jobId := awsutils.InventoryTowElementsOfMappingVault(restorationContext)
	loggers.Printfln(loggers.OptionalInfo, "Job to find mapping archive id has started (can last up to 4 hours): %s", jobId)
	restorationContext.RegionVaultCache.MappingVaultInventoryJobId = jobId
	restorationContext.WriteRegionVaultCache()
	return jobId
}

