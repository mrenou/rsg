package awsutils

import (
	"time"
	"github.com/aws/aws-sdk-go/aws"
	"../loggers"
	"../utils"
	"github.com/aws/aws-sdk-go/service/glacier"
	"strconv"
	"errors"
	"io"
	"os"
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
	loggers.Printf(loggers.Debug, "describe job with params: %v\n", params)
	return restorationContext.GlacierClient.DescribeJob(params)
}

func JobIsCompleted(restorationContext *RestorationContext, vault, jobId string) (bool, error) {
	if resp, err := DescribeJob(restorationContext, vault, jobId); err == nil {
		return *resp.Completed, nil
	} else {
		return false, err
	}
}

func DownloadArchiveTo(restorationContext *RestorationContext, vault, jobId string, filename string) {
	DownloadPartialArchiveTo(restorationContext, vault, jobId, filename, 0, 0)
}

func DownloadPartialArchiveTo(restorationContext *RestorationContext, vault, jobId string, destPath string, fromByte, sizeToDownload uint64) {
	var rangeToRetrieve *string = nil
	if sizeToDownload != 0 {
		rangeToRetrieve = aws.String(strconv.FormatUint(fromByte, 10) + "-" + strconv.FormatUint(fromByte + sizeToDownload - 1, 10))
	}
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
		Range: rangeToRetrieve,
	}
	loggers.Printf(loggers.Debug, "get output job with params: %v\n", params)
	resp, err := restorationContext.GlacierClient.GetJobOutput(params)
	utils.ExitIfError(err)
	defer resp.Body.Close()
	var file *os.File;
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		file, err = os.Create(destPath)
	} else {
		file, err = os.OpenFile(destPath, os.O_APPEND | os.O_WRONLY, 0600)
	}
	utils.ExitIfError(err)
	loggers.Printf(loggers.Debug, "copy file into: %v\n", destPath)
	written, err := io.Copy(file, resp.Body)
	loggers.Printf(loggers.Debug, "%v bytes copied\n", written)
	utils.ExitIfError(err)
}

func StartRetrieveArchiveJob(restorationContext *RestorationContext, vault string, archive Archive) string {
	jobId, _ := StartRetrievePartialArchiveJob(restorationContext, vault, archive, 0, archive.Size)
	return jobId
}

func StartRetrievePartialArchiveJob(restorationContext *RestorationContext, vault string, archive Archive, fromByte uint64, sizeToRetrieve uint64) (string, uint64) {
	rangeToRetrieve := ""
	if (fromByte) % utils.S_1MB != 0 {
		utils.ExitIfError(errors.New("byte start index must be divisible by 1MB"))
	}
	if (fromByte + sizeToRetrieve >= archive.Size) {
		sizeToRetrieve = archive.Size - fromByte
	} else if sizeToRetrieve >= utils.S_1MB {
		sizeToRetrieve = sizeToRetrieve - (sizeToRetrieve % utils.S_1MB)
	} else {
		utils.ExitIfError(errors.New("size to retrieve must be divisible by MB"))
	}
	rangeToRetrieve = strconv.FormatUint(fromByte, 10) + "-" + strconv.FormatUint(fromByte + sizeToRetrieve - 1, 10)

	params := &glacier.InitiateJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		VaultName: aws.String(vault),
		JobParameters: &glacier.JobParameters{
			ArchiveId: aws.String(archive.ArchiveId),
			//Description: aws.String("restore mapping from " + restorationContext.MappingVault),
			Type:        aws.String("archive-retrieval"),
			RetrievalByteRange: aws.String(rangeToRetrieve),
		},
	}
	loggers.Printf(loggers.Debug, "start retrieve archive job with params: %v\n", params)
	resp, err := restorationContext.GlacierClient.InitiateJob(params)
	utils.ExitIfError(err)
	return *resp.JobId, sizeToRetrieve
}