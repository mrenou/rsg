package awsutils

import (
	"time"
	"github.com/aws/aws-sdk-go/aws"
	"../loggers"
	"../utils"
	"github.com/aws/aws-sdk-go/service/glacier"
	"io/ioutil"
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
	loggers.DebugPrintf("describe job %s on vault %s\n", jobId, restorationContext.Vault)
	params := &glacier.DescribeJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
	}
	return restorationContext.GlacierClient.DescribeJob(params)
}

func JobIsCompleted(restorationContext *RestorationContext, vault, jobId string) (bool, error) {
	if resp, err := DescribeJob(restorationContext, vault, jobId); err == nil {
		return *resp.Completed, nil
	} else {
		return false, err
	}
}

func DownloadArchiveTo(restorationContext *RestorationContext, vault , jobId string, filename string) {
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
	}
	resp, err := restorationContext.GlacierClient.GetJobOutput(params)
	utils.ExitIfError(err)
	bytes, err := ioutil.ReadAll(resp.Body)
	utils.ExitIfError(err)
	defer resp.Body.Close()
	err = ioutil.WriteFile(filename, bytes, 0600)
	utils.ExitIfError(err)
}

func StartRetrieveArchiveJob(restorationContext *RestorationContext, vault string, archive Archive) string {
	var rangeToRetrive *string = nil
	// TODO use rangeToRetrive when limit download
	params := &glacier.InitiateJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		VaultName: aws.String(vault),
		JobParameters: &glacier.JobParameters{
			ArchiveId: aws.String(archive.ArchiveId),
			Description: aws.String("restore mapping from " + restorationContext.MappingVault),
			Type:        aws.String("archive-retrieval"),
			RetrievalByteRange: rangeToRetrive,
		},
	}
	resp, err := restorationContext.GlacierClient.InitiateJob(params)
	utils.ExitIfError(err)
	restorationContext.RegionVaultCache.MappingVaultRetrieveJobId = *resp.JobId
	restorationContext.WriteRegionVaultCache()
	return *resp.JobId
}