package awsutils

import (
	"time"
	"github.com/aws/aws-sdk-go/aws"
	"rsg/outputs"
	"rsg/utils"
	"github.com/aws/aws-sdk-go/service/glacier"
	"strconv"
	"errors"
	"io"
	"os"
	"io/ioutil"
	"encoding/json"
	"code.cloudfoundry.org/bytefmt"
	"github.com/aws/aws-sdk-go/service/glacier/glacieriface"
	"strings"
)

type jobIdsAtStartupStruct struct {
	fileRetrievalJobIdByRangeByArchiveId map[string]map[string]string
	MappingInventoryJobId                string
	MappingRetrievalJobId                string
}

type Archive struct {
	ArchiveId string
	Size      uint64
}

var WaitTime = 5 * time.Minute
var JobIdsAtStartup = &jobIdsAtStartupStruct{fileRetrievalJobIdByRangeByArchiveId: make(map[string]map[string]string)}

// for test
func AddRetrievalJobAtStartup(archiveId, retrievalByteRange, jobId string) {
	fileRetrievalJobIdByRange, ok := JobIdsAtStartup.fileRetrievalJobIdByRangeByArchiveId[archiveId]
	if !ok {
		fileRetrievalJobIdByRange = make(map[string]string)
		JobIdsAtStartup.fileRetrievalJobIdByRangeByArchiveId[archiveId] = fileRetrievalJobIdByRange
	}
	fileRetrievalJobIdByRange[retrievalByteRange] = jobId
}

func LoadJobIdsAtStartup(glacierClient glacieriface.GlacierAPI, mappingVault, vault string) {
	fileRetrievalJobCounter := 0
	recordJobsFn := func(page *glacier.ListJobsOutput, lastPage bool) bool {
		for _, desc := range page.JobList {
			if *desc.StatusCode == "InProgress" || *desc.StatusCode == "Succeeded" {
				if *desc.Action == "ArchiveRetrieval" {
					if strings.HasSuffix(*desc.VaultARN, "_mapping") {
						JobIdsAtStartup.MappingRetrievalJobId = *desc.JobId
					} else {
						fileRetrievalJobIdByRange, ok := JobIdsAtStartup.fileRetrievalJobIdByRangeByArchiveId[*desc.ArchiveId]
						if !ok {
							fileRetrievalJobIdByRange = make(map[string]string)
							JobIdsAtStartup.fileRetrievalJobIdByRangeByArchiveId[*desc.ArchiveId] = fileRetrievalJobIdByRange
						}
						if desc.RetrievalByteRange != nil {
							fileRetrievalJobIdByRange[*desc.RetrievalByteRange] = *desc.JobId
							fileRetrievalJobCounter++
						}
					}
				} else {
					if strings.HasSuffix(*desc.VaultARN, "_mapping") {
						JobIdsAtStartup.MappingInventoryJobId = *desc.JobId
					}
				}
			}
		}
		return true
	}
	DoOnJobPages(glacierClient, mappingVault, recordJobsFn)
	DoOnJobPages(glacierClient, vault, recordJobsFn)
	if JobIdsAtStartup.MappingInventoryJobId != "" {
		outputs.Printfln(outputs.Verbose, "Mapping inventory job found : %s", JobIdsAtStartup.MappingInventoryJobId)
	}
	if JobIdsAtStartup.MappingRetrievalJobId != "" {
		outputs.Printfln(outputs.Verbose, "Mapping retrivial job found : %s", JobIdsAtStartup.MappingRetrievalJobId)
	}
	if fileRetrievalJobCounter > 0 {
		outputs.Printfln(outputs.Verbose, "%v file retrivial job found", fileRetrievalJobCounter)
	}
}

func (jobIdsAtStartup *jobIdsAtStartupStruct) GetJobIdForFileRetrieval(archiveId, retrievalByteRange string) string {
	if fileRetrievalJobIdByRange, ok := jobIdsAtStartup.fileRetrievalJobIdByRangeByArchiveId[archiveId]; ok {
		if fileRetrievalJobId, ok := fileRetrievalJobIdByRange[retrievalByteRange]; ok {
			return fileRetrievalJobId
		}
	}
	return ""
}

func WaitJobIsCompleted(glacierClient glacieriface.GlacierAPI, vault, jobId string) {
	for {
		completed, err := JobIsCompleted(glacierClient, vault, jobId)
		if completed {
			return
		} else if err != nil {
			utils.ExitIfError(err)
		}
		time.Sleep(1 * WaitTime)
	}
}

func JobIsCompleted(glacierClient glacieriface.GlacierAPI, vault, jobId string) (bool, error) {
	if resp, err := DescribeJob(glacierClient, vault, jobId); err == nil {
		return *resp.Completed, nil
	} else {
		return false, err
	}
}

func DescribeJob(glacierClient glacieriface.GlacierAPI, vault, jobId string) (*glacier.JobDescription, error) {
	params := &glacier.DescribeJobInput{
		AccountId: aws.String(AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
	}
	outputs.Printfln(outputs.Verbose, "Aws call: glacier.DescribeJob(%+v)", params)
	resp, err := glacierClient.DescribeJob(params)
	outputs.Printfln(outputs.Verbose, "Aws response: %v (error %v)\n", resp, err)
	return resp, err
}

func DownloadArchiveTo(glacierClient glacieriface.GlacierAPI, vault, jobId string, filename string) uint64 {
	return DownloadPartialArchiveTo(glacierClient, vault, jobId, filename, 0, 0, 0)
}

func DownloadPartialArchiveTo(glacierClient glacieriface.GlacierAPI, vault, jobId, destPath string, fromByteToDownload, sizeToDownload, fromByteToWrite uint64) uint64 {
	var rangeToRetrieve *string = nil
	if sizeToDownload != 0 {
		rangeToRetrieve = aws.String(strconv.FormatUint(fromByteToDownload, 10) + "-" + strconv.FormatUint(fromByteToDownload + sizeToDownload - 1, 10))
	}
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
		Range: rangeToRetrieve,
	}
	outputs.Printfln(outputs.Verbose, "Aws call: glacier.GetJobOutput(%v)", params)
	resp, err := glacierClient.GetJobOutput(params)
	outputs.Printfln(outputs.Verbose, "Aws response: %v (error %v)\n", resp, err)
	utils.ExitIfError(err)
	defer resp.Body.Close()
	var file *os.File;
	file, err = os.OpenFile(destPath, os.O_CREATE | os.O_RDWR, 0600)
	file.Seek(int64(fromByteToWrite), os.SEEK_SET)
	utils.ExitIfError(err)
	outputs.Printfln(outputs.Verbose, "Copy file into: %v", destPath)
	written, err := io.Copy(file, resp.Body)
	written64 := uint64(written)
	outputs.Printfln(outputs.Verbose, "%v copied", bytefmt.ByteSize(written64))
	utils.ExitIfError(err)
	return written64
}

type JobStartStatus struct {
	JobId         string
	IsResumed     bool
	IsSuccess     bool
	Err           error
	SizeRetrieved uint64
}

func StartRetrieveArchiveJob(glacierClient glacieriface.GlacierAPI, vault string, archive Archive) JobStartStatus {
	return StartRetrievePartialArchiveJob(glacierClient, vault, archive, 0, archive.Size)
}

func StartRetrievePartialArchiveJob(glacierClient glacieriface.GlacierAPI, vault string, archive Archive, fromByte uint64, sizeToRetrieve uint64) JobStartStatus {
	rangeToRetrieve := ""
	if (fromByte) % utils.S_1MB != 0 {
		return JobStartStatus{IsSuccess: false, Err:  errors.New("Byte start index must be divisible by 1MB")}
	}
	if (fromByte + sizeToRetrieve >= archive.Size) {
		sizeToRetrieve = archive.Size - fromByte
	} else if sizeToRetrieve >= utils.S_1MB {
		sizeToRetrieve = sizeToRetrieve - (sizeToRetrieve % utils.S_1MB)
	} else {
		return JobStartStatus{IsSuccess: false, Err: errors.New("Size to retrieve must be divisible by MB")}
	}
	rangeToRetrieve = strconv.FormatUint(fromByte, 10) + "-" + strconv.FormatUint(fromByte + sizeToRetrieve - 1, 10)

	if existingJobsId := JobIdsAtStartup.GetJobIdForFileRetrieval(archive.ArchiveId, rangeToRetrieve); existingJobsId != "" {
		return JobStartStatus{JobId: existingJobsId, IsResumed: true, IsSuccess: true, SizeRetrieved: sizeToRetrieve}
	} else {
		params := &glacier.InitiateJobInput{
			AccountId: aws.String(AccountId),
			VaultName: aws.String(vault),
			JobParameters: &glacier.JobParameters{
				ArchiveId: aws.String(archive.ArchiveId),
				Type:        aws.String("archive-retrieval"),
				RetrievalByteRange: aws.String(rangeToRetrieve),
			},
		}
		outputs.Printfln(outputs.Verbose, "Aws call: glacier.InitiateJob(%v)", params)
		resp, err := glacierClient.InitiateJob(params)
		outputs.Printfln(outputs.Verbose, "Aws response: %v (error %v)\n", resp, err)
		if err != nil {
			return JobStartStatus{IsSuccess: false, Err: err}
		}
		return JobStartStatus{JobId: *resp.JobId, IsResumed: false, IsSuccess: true, SizeRetrieved: sizeToRetrieve}
	}
}

func GetDataRetrievalStrategy(glacierClient glacieriface.GlacierAPI) string {
	params := &glacier.GetDataRetrievalPolicyInput{
		AccountId:  &AccountId,
	}
	outputs.Printfln(outputs.Verbose, "Aws call: glacier.GetDataRetrievalPolicy(%v)", params)
	resp, err := glacierClient.GetDataRetrievalPolicy(params)
	outputs.Printfln(outputs.Verbose, "Aws response: %v (error %v)\n", resp, err)
	utils.ExitIfError(err)
	return *resp.Policy.Rules[0].Strategy
}

func InventoryTowElementsOfVault(glacierClient glacieriface.GlacierAPI, vault string) string {
	params := &glacier.InitiateJobInput{
		AccountId: aws.String(AccountId),
		VaultName: aws.String(vault),
		JobParameters: &glacier.JobParameters{
			Type:        aws.String("inventory-retrieval"),
			InventoryRetrievalParameters: &glacier.InventoryRetrievalJobInput{Limit: aws.String("2")},
		},
	}
	outputs.Printfln(outputs.Verbose, "Aws call: glacier.InitiateJob(%v)", params)
	resp, err := glacierClient.InitiateJob(params)
	outputs.Printfln(outputs.Verbose, "Aws response: %v (error %v)\n", resp, err)
	utils.ExitIfError(err)
	return *(resp.JobId)
}

type VaultInventory struct {
	ArchiveList []Archive
}

func GetArchiveIdFromInventory(glacierClient glacieriface.GlacierAPI, vault, jobId string) *Archive {
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
		Range:     nil,
	}
	outputs.Printfln(outputs.Verbose, "Aws call: glacier.GetJobOutput(%v)", params)
	resp, err := glacierClient.GetJobOutput(params)
	outputs.Printfln(outputs.Verbose, "Aws response: %v (error %v)\n", resp, err)
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

func DoOnJobPages(glacierClient glacieriface.GlacierAPI, vault string, fn func(*glacier.ListJobsOutput, bool) bool) {
	params := &glacier.ListJobsInput{
		AccountId: aws.String(AccountId),
		VaultName: aws.String(vault),
	}
	outputs.Printfln(outputs.Verbose, "Aws call: glacier.ListJobsPages(%v)", params)
	err := glacierClient.ListJobsPages(params, fn)
	outputs.Printfln(outputs.Verbose, "Aws error %v\n", err)
	utils.ExitIfError(err)
}