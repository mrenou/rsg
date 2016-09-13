package awsutils

import (
	"time"
	"github.com/aws/aws-sdk-go/aws"
	"rsg/loggers"
	"rsg/utils"
	"github.com/aws/aws-sdk-go/service/glacier"
	"strconv"
	"errors"
	"io"
	"os"
	"io/ioutil"
	"encoding/json"
	"code.cloudfoundry.org/bytefmt"
)

var WaitTime = 5 * time.Minute

func WaitJobIsCompleted(restorationContext *RestorationContext, vault, jobId string) {
	for {
		time.Sleep(1 * WaitTime)
		completed, err := JobIsCompleted(restorationContext, vault, jobId)
		if completed {
			return
		} else if err != nil {
			utils.ExitIfError(err)
		}
	}
}

func DescribeJob(restorationContext *RestorationContext, vault, jobId string) (*glacier.JobDescription, error) {
	params := &glacier.DescribeJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
	}
	loggers.Printfln(loggers.Verbose, "Aws call: glacier.DescribeJob(%+v)", params)
	return restorationContext.GlacierClient.DescribeJob(params)
}

func JobIsCompleted(restorationContext *RestorationContext, vault, jobId string) (bool, error) {
	if resp, err := DescribeJob(restorationContext, vault, jobId); err == nil {
		return *resp.Completed, nil
	} else {
		return false, err
	}
}

func DownloadArchiveTo(restorationContext *RestorationContext, vault, jobId string, filename string) uint64 {
	return DownloadPartialArchiveTo(restorationContext, vault, jobId, filename, 0, 0, 0)
}

func DownloadPartialArchiveTo(restorationContext *RestorationContext, vault, jobId string, destPath string, fromByteToDownload, sizeToDownload, fromByteToWrite uint64) uint64 {
	var rangeToRetrieve *string = nil
	if sizeToDownload != 0 {
		rangeToRetrieve = aws.String(strconv.FormatUint(fromByteToDownload, 10) + "-" + strconv.FormatUint(fromByteToDownload + sizeToDownload - 1, 10))
	}
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
		Range: rangeToRetrieve,
	}
	loggers.Printfln(loggers.Verbose, "Aws call: glacier.GetJobOutput(%v)", params)
	resp, err := restorationContext.GlacierClient.GetJobOutput(params)
	utils.ExitIfError(err)
	defer resp.Body.Close()
	var file *os.File;
	file, err = os.OpenFile(destPath, os.O_CREATE | os.O_RDWR, 0600)
	file.Seek(int64(fromByteToWrite), os.SEEK_SET)
	utils.ExitIfError(err)
	loggers.Printfln(loggers.Verbose, "Copy file into: %v", destPath)
	written, err := io.Copy(file, resp.Body)
	written64 := uint64(written)
	loggers.Printfln(loggers.Verbose, "%v copied", bytefmt.ByteSize(written64))
	utils.ExitIfError(err)
	return written64
}

func StartRetrieveArchiveJob(restorationContext *RestorationContext, vault string, archive Archive) (string, uint64, error) {
	return StartRetrievePartialArchiveJob(restorationContext, vault, archive, 0, archive.Size)
}

func StartRetrievePartialArchiveJob(restorationContext *RestorationContext, vault string, archive Archive, fromByte uint64, sizeToRetrieve uint64) (string, uint64, error) {
	rangeToRetrieve := ""
	if (fromByte) % utils.S_1MB != 0 {
		return "", 0, errors.New("Byte start index must be divisible by 1MB")
	}
	if (fromByte + sizeToRetrieve >= archive.Size) {
		sizeToRetrieve = archive.Size - fromByte
	} else if sizeToRetrieve >= utils.S_1MB {
		sizeToRetrieve = sizeToRetrieve - (sizeToRetrieve % utils.S_1MB)
	} else {
		return "", 0, errors.New("Size to retrieve must be divisible by MB")
	}
	rangeToRetrieve = strconv.FormatUint(fromByte, 10) + "-" + strconv.FormatUint(fromByte + sizeToRetrieve - 1, 10)
	params := &glacier.InitiateJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		VaultName: aws.String(vault),
		JobParameters: &glacier.JobParameters{
			ArchiveId: aws.String(archive.ArchiveId),
			Type:        aws.String("archive-retrieval"),
			RetrievalByteRange: aws.String(rangeToRetrieve),
		},
	}
	loggers.Printfln(loggers.Verbose, "Aws call: glacier.InitiateJob(%v)", params)
	resp, err := restorationContext.GlacierClient.InitiateJob(params)
	if err != nil {
		return "", 0, err
	}
	return *resp.JobId, sizeToRetrieve, nil
}

func GetDataRetrievalStrategy(restorationContext *RestorationContext) string {
	params := &glacier.GetDataRetrievalPolicyInput{
		AccountId:  &restorationContext.AccountId,
	}
	loggers.Printfln(loggers.Verbose, "Aws call: glacier.GetDataRetrievalPolicy(%v)", params)
	output, err := restorationContext.GlacierClient.GetDataRetrievalPolicy(params)
	utils.ExitIfError(err)
	return *output.Policy.Rules[0].Strategy
}

func InventoryTowElementsOfMappingVault(restorationContext *RestorationContext) string {
	params := &glacier.InitiateJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		VaultName: aws.String(restorationContext.MappingVault),
		JobParameters: &glacier.JobParameters{
			Description: aws.String("inventory " + restorationContext.MappingVault),
			Type:        aws.String("inventory-retrieval"),
			InventoryRetrievalParameters: &glacier.InventoryRetrievalJobInput{Limit: aws.String("2")},
		},
	}
	loggers.Printfln(loggers.Verbose, "Aws call: glacier.InitiateJob(%v)", params)
	resp, err := restorationContext.GlacierClient.InitiateJob(params)
	utils.ExitIfError(err)
	return *(resp.JobId)
}

type VaultInventory struct {
	ArchiveList []Archive
}

func GetMappingArchiveIdFromInventory(restorationContext *RestorationContext, jobId string) *Archive {
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(restorationContext.MappingVault),
		Range:     nil,
	}
	loggers.Printfln(loggers.Verbose, "Aws call: glacier.GetJobOutput(%v)", params)
	resp, err := restorationContext.GlacierClient.GetJobOutput(params)
	utils.ExitIfError(err)
	defer resp.Body.Close()
	jsonContent, _ := ioutil.ReadAll(resp.Body)
	vaultInventory := VaultInventory{}
	err = json.Unmarshal(jsonContent, &vaultInventory)
	utils.ExitIfError(err)
	if (len(vaultInventory.ArchiveList) != 1) {
		utils.ExitIfError(errors.New("Mapping vault shoud be have only one archive"))
	}
	return &vaultInventory.ArchiveList[0]
}