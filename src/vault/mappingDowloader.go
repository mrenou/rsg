package vault

import (
	"../loggers"
	"../utils"
	"../inputs"
	"../awsutils"
	"strings"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/aws"
	"io/ioutil"
	"encoding/json"
	"errors"
	"os"
	"fmt"
)

func DownloadMappingArchive(restorationContext *awsutils.RestorationContext) {
	if stat, err := os.Stat(restorationContext.GetMappingFilePath()); os.IsNotExist(err) {
		downloadMappingArchive(restorationContext)
	} else {
		if inputs.QueryYesOrNo(fmt.Sprintf("local mapping archive already exists with last modification date %v, retrieve a new mapping file ?", stat.ModTime().Format("Mon Jan _2 15:04:05 2006")), false) {
			os.Remove(restorationContext.GetMappingFilePath())
			downloadMappingArchive(restorationContext)
		}
	}

}

func downloadMappingArchive(restorationContext *awsutils.RestorationContext) {
	jobId, jobCompleted := checkRetrieveMappingOrStartNewJob(restorationContext, getMappingArchive(restorationContext))
	if !jobCompleted {
		awsutils.WaitJobIsCompleted(restorationContext, restorationContext.MappingVault, jobId)
		loggers.Printf(loggers.Info, "job has finished: %s\n", jobId)
	}
	awsutils.DownloadArchiveTo(restorationContext, restorationContext.MappingVault, jobId, restorationContext.GetMappingFilePath())
	restorationContext.RegionVaultCache.MappingArchive = nil
	restorationContext.RegionVaultCache.MappingVaultRetrieveJobId = ""
	restorationContext.WriteRegionVaultCache()
	loggers.Print(loggers.Info, "mapping archive has been downloaded\n")
}

func checkRetrieveMappingOrStartNewJob(restorationContext *awsutils.RestorationContext, archive awsutils.Archive) (string, bool) {
	jobCompleted := false
	jobId := restorationContext.RegionVaultCache.MappingVaultRetrieveJobId
	var err error;
	if jobId != "" {
		loggers.Printf(loggers.Debug, "retrieve mapping archive job id found : %s\n", jobId)
		jobCompleted, err = awsutils.JobIsCompleted(restorationContext, restorationContext.MappingVault, restorationContext.RegionVaultCache.MappingVaultRetrieveJobId)
		if jobCompleted == false {
			if err == nil {
				loggers.Printf(loggers.Info, "job to retrieve mapping archive is in progress (can last up to 4 hours): %s\n", jobId)
			} else if strings.Contains(err.Error(), "The job ID was not found") {
				loggers.Print(loggers.Warning, "retrieve mapping archive job cached was not found\n")
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
	jobId := awsutils.StartRetrieveArchiveJob(restorationContext, restorationContext.MappingVault, archive)
	restorationContext.RegionVaultCache.MappingVaultRetrieveJobId = jobId
	restorationContext.WriteRegionVaultCache()
	loggers.Printf(loggers.Info, "job to retrieve mapping archive has started (can last up to 4 hours): %s\n", jobId)
	return jobId
}

func getMappingArchive(restorationContext *awsutils.RestorationContext) awsutils.Archive {
	mappingArchive := restorationContext.RegionVaultCache.MappingArchive
	if mappingArchive == nil {
		jobId, jobCompleted := checkMappingInventoryOrStartNewJob(restorationContext)
		if jobCompleted == false {
			awsutils.WaitJobIsCompleted(restorationContext, restorationContext.MappingVault, jobId)
			loggers.Printf(loggers.Info, "job has finished: %s\n", jobId)
		}
		restorationContext.RegionVaultCache.MappingVaultInventoryJobId = ""
		restorationContext.RegionVaultCache.MappingArchive = getMappingArchiveIdFromInventory(restorationContext, jobId)
		restorationContext.WriteRegionVaultCache()
	}
	loggers.Printf(loggers.Debug, "Mapping archive id is %s\n", restorationContext.RegionVaultCache.MappingArchive.ArchiveId)
	return *restorationContext.RegionVaultCache.MappingArchive
}

func checkMappingInventoryOrStartNewJob(restorationContext *awsutils.RestorationContext) (string, bool) {
	jobCompleted := false
	jobId := restorationContext.RegionVaultCache.MappingVaultInventoryJobId
	var err error
	if jobId != "" {
		loggers.Printf(loggers.Debug, "mapping vault inventory job id found : %s\n", jobId)
		jobCompleted, err = awsutils.JobIsCompleted(restorationContext, restorationContext.MappingVault, jobId)
		if jobCompleted == false {
			if err == nil {
				loggers.Printf(loggers.Info, "job to find mapping archive id is in progress (can last up to 4 hours): %s\n", jobId)
			} else if strings.Contains(err.Error(), "The job ID was not found") {
				loggers.Print(loggers.Warning, "inventory job cahed for mapping vaul was not found\n")
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
	jobId := inventoryTowElementsOfMappingVault(restorationContext)
	loggers.Printf(loggers.Info, "job to find mapping archive id has started (can last up to 4 hours): %s\n", jobId)
	restorationContext.RegionVaultCache.MappingVaultInventoryJobId = jobId
	restorationContext.WriteRegionVaultCache()
	return jobId
}

func inventoryTowElementsOfMappingVault(restorationContext *awsutils.RestorationContext) string {
	params := &glacier.InitiateJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		VaultName: aws.String(restorationContext.MappingVault),
		JobParameters: &glacier.JobParameters{
			Description: aws.String("inventory " + restorationContext.MappingVault),
			Type:        aws.String("inventory-retrieval"),
			InventoryRetrievalParameters: &glacier.InventoryRetrievalJobInput{Limit: aws.String("2")},
		},
	}
	resp, err := restorationContext.GlacierClient.InitiateJob(params)
	utils.ExitIfError(err)
	return *(resp.JobId)
}

type VaultInventory struct {
	ArchiveList []awsutils.Archive
}

func getMappingArchiveIdFromInventory(restorationContext *awsutils.RestorationContext, jobId string) *awsutils.Archive {
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(restorationContext.MappingVault),
		Range:     nil,
	}
	resp, err := restorationContext.GlacierClient.GetJobOutput(params)
	utils.ExitIfError(err)
	defer resp.Body.Close()
	jsonContent, _ := ioutil.ReadAll(resp.Body)
	vaultInventory := VaultInventory{}
	err = json.Unmarshal(jsonContent, &vaultInventory)
	utils.ExitIfError(err)
	if (len(vaultInventory.ArchiveList) != 1) {
		utils.ExitIfError(errors.New("mapping vault shoud be have only one archive"))
	}
	return &vaultInventory.ArchiveList[0]
}
