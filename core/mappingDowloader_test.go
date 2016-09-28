package core

import (
	"rsg/awsutils"
	"rsg/inputs"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/aws"
	"testing"
	"bytes"
	"io"
	"io/ioutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"errors"
	"bufio"
	"regexp"
	"strings"
	"rsg/consts"
)

func mockStartMappingJobInventory(glacierMock *GlacierMock, vault string) *mock.Call {
	params := &glacier.InitiateJobInput{
		AccountId: aws.String(awsutils.AccountId),
		VaultName: aws.String(vault),
		JobParameters: &glacier.JobParameters{
			Type:        aws.String("inventory-retrieval"),
			InventoryRetrievalParameters: &glacier.InventoryRetrievalJobInput{Limit: aws.String("2")},
		},
	}

	out := &glacier.InitiateJobOutput{
		JobId: aws.String("inventoryMappingJobId"),
	}

	return glacierMock.On("InitiateJob", params).Return(out, nil)
}

func mockDescribeJob(glacierMock *GlacierMock, jobId, vault string, completed bool) *mock.Call {
	params := &glacier.DescribeJobInput{
		AccountId: aws.String(awsutils.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
	}

	out := &glacier.JobDescription{
		Completed: aws.Bool(completed),
	}

	return glacierMock.On("DescribeJob", params).Return(out, nil)
}

func mockDescribeJobErr(glacierMock *GlacierMock, jobId, vault string, err error) *mock.Call {
	params := &glacier.DescribeJobInput{
		AccountId: aws.String(awsutils.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
	}

	return glacierMock.On("DescribeJob", params).Return(nil, err)
}

func mockOutputJob(glacierMock *GlacierMock, jobId, vault string, content []byte) *mock.Call {
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(awsutils.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
	}

	out := &glacier.GetJobOutputOutput{
		Body:  newReaderClosable(bytes.NewReader(content)),
	}

	return glacierMock.On("GetJobOutput", params).Return(out, nil)
}

func mockStartRetrieveJob(glacierMock *GlacierMock, vault, archiveId, retrievalByteRange, jobIdToReturn string) *mock.Call {
	params := &glacier.InitiateJobInput{
		AccountId: aws.String(awsutils.AccountId),
		VaultName: aws.String(vault),
		JobParameters: &glacier.JobParameters{
			ArchiveId: aws.String(archiveId),
			Type:        aws.String("archive-retrieval"),
			RetrievalByteRange: aws.String(retrievalByteRange),
		},
	}

	out := &glacier.InitiateJobOutput{
		JobId: aws.String(jobIdToReturn),
	}
	return glacierMock.On("InitiateJob", params).Return(out, nil)
}

func TestDownloadMappingArchive_download_mapping_first_time(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	glacierMock, restorationContext := InitTestWithGlacier()

	mockStartMappingJobInventory(glacierMock, restorationContext.MappingVault)
	mockDescribeJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, []byte("{\"ArchiveList\":[{\"ArchiveId\":\"mappingArchiveId\",\"Size\":42}]}"))
	mockStartRetrieveJob(glacierMock, restorationContext.MappingVault, "mappingArchiveId", "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "Job to find mapping archive id has started (can last up to 4 hours): inventoryMappingJobId" + consts.LINE_BREAK +
		"Job has finished: inventoryMappingJobId" + consts.LINE_BREAK +
		"Job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId" + consts.LINE_BREAK +
		"Job has finished: retrieveMappingJobId" + consts.LINE_BREAK +
		"Mapping archive has been downloaded" + consts.LINE_BREAK, string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_inventory_job_in_progress(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	glacierMock, restorationContext := InitTestWithGlacier()
	awsutils.JobIdsAtStartup.MappingInventoryJobId = "inventoryMappingJobId"

	mockDescribeJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, false).Once()
	mockDescribeJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, []byte("{\"ArchiveList\":[{\"ArchiveId\":\"mappingArchiveId\",\"Size\":42}]}"))
	mockStartRetrieveJob(glacierMock, restorationContext.MappingVault, "mappingArchiveId", "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	//Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "Job to find mapping archive id is in progress (can last up to 4 hours): inventoryMappingJobId" + consts.LINE_BREAK +
		"Job has finished: inventoryMappingJobId" + consts.LINE_BREAK +
		"Job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId" + consts.LINE_BREAK +
		"Job has finished: retrieveMappingJobId" + consts.LINE_BREAK +
		"Mapping archive has been downloaded" + consts.LINE_BREAK, string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_inventory_job_deprecated(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	glacierMock, restorationContext := InitTestWithGlacier()
	awsutils.JobIdsAtStartup.MappingInventoryJobId = "unknownInventoryMappingJobId"

	mockDescribeJobErr(glacierMock, "unknownInventoryMappingJobId", restorationContext.MappingVault, errors.New("The job ID was not found"))
	mockStartMappingJobInventory(glacierMock, restorationContext.MappingVault)
	mockDescribeJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, []byte("{\"ArchiveList\":[{\"ArchiveId\":\"mappingArchiveId\",\"Size\":42}]}"))
	mockStartRetrieveJob(glacierMock, restorationContext.MappingVault, "mappingArchiveId", "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "WARNING: Inventory job cahed for mapping vaul was not found" + consts.LINE_BREAK +
		"Job to find mapping archive id has started (can last up to 4 hours): inventoryMappingJobId" + consts.LINE_BREAK +
		"Job has finished: inventoryMappingJobId" + consts.LINE_BREAK +
		"Job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId" + consts.LINE_BREAK +
		"Job has finished: retrieveMappingJobId" + consts.LINE_BREAK +
		"Mapping archive has been downloaded" + consts.LINE_BREAK, string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_inventory_done(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	glacierMock, restorationContext := InitTestWithGlacier()
	awsutils.JobIdsAtStartup.MappingInventoryJobId = "inventoryMappingJobId"

	mockDescribeJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, []byte("{\"ArchiveList\":[{\"ArchiveId\":\"mappingArchiveId\",\"Size\":42}]}"))
	mockStartRetrieveJob(glacierMock, restorationContext.MappingVault, "mappingArchiveId", "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "Job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId" + consts.LINE_BREAK +
		"Job has finished: retrieveMappingJobId" + consts.LINE_BREAK +
		"Mapping archive has been downloaded" + consts.LINE_BREAK, string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_retrieve_job_in_progress(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	glacierMock, restorationContext := InitTestWithGlacier()
	awsutils.JobIdsAtStartup.MappingRetrievalJobId = "retrieveMappingJobId"
	restorationContext.RegionVaultCache = RegionVaultCache{MappingArchive: &awsutils.Archive{"mappingArchiveId", 42},}

	mockDescribeJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, false).Once()
	mockDescribeJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	//Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "Job to retrieve mapping archive is in progress (can last up to 4 hours): retrieveMappingJobId" + consts.LINE_BREAK +
		"Job has finished: retrieveMappingJobId" + consts.LINE_BREAK +
		"Mapping archive has been downloaded" + consts.LINE_BREAK, string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_retrieve_job_deprecated(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	glacierMock, restorationContext := InitTestWithGlacier()
	awsutils.JobIdsAtStartup.MappingRetrievalJobId = "unknownRetrieveMappingJobId"
	restorationContext.RegionVaultCache = RegionVaultCache{MappingArchive: &awsutils.Archive{"mappingArchiveId", 42},}

	mockDescribeJobErr(glacierMock, "unknownRetrieveMappingJobId", restorationContext.MappingVault, errors.New("The job ID was not found"))
	mockStartRetrieveJob(glacierMock, restorationContext.MappingVault, "mappingArchiveId", "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "WARNING: Retrieve mapping archive job cached was not found" + consts.LINE_BREAK +
		"Job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId" + consts.LINE_BREAK +
		"Job has finished: retrieveMappingJobId" + consts.LINE_BREAK +
		"Mapping archive has been downloaded" + consts.LINE_BREAK, string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_retrieve_done(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	glacierMock, restorationContext := InitTestWithGlacier()
	awsutils.JobIdsAtStartup.MappingRetrievalJobId = "retrieveMappingJobId"
	restorationContext.RegionVaultCache = RegionVaultCache{MappingArchive: &awsutils.Archive{"mappingArchiveId", 42},}

	mockDescribeJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "Mapping archive has been downloaded" + consts.LINE_BREAK, string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_mapping_already_exists(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	awsutils.JobIdsAtStartup.MappingRetrievalJobId = "retrieveMappingJobId"
	restorationContext.RegionVaultCache = RegionVaultCache{MappingArchive: &awsutils.Archive{"mappingArchiveId", 42},}

	ioutil.WriteFile("../../testtmp/cache/mapping.sqllite", []byte("hello !"), 0600)

	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte(consts.LINE_BREAK)))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	glacierMock.AssertNotCalled(t, mock.Anything)

	r := regexp.MustCompile("Local mapping archive already exists with last modification date .+, retrieve a new mapping file \\?\\[y/N\\]")
	assert.True(t, r.Match(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_mapping_already_exists_but_restart_download(t *testing.T) {
	// Given
	buffer := CommonInitTest()
	glacierMock, restorationContext := InitTestWithGlacier()

	ioutil.WriteFile("../../testtmp/cache/mapping.sqllite", []byte("hello !"), 0600)

	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("y" + consts.LINE_BREAK)))

	mockStartMappingJobInventory(glacierMock, restorationContext.MappingVault)
	mockDescribeJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "inventoryMappingJobId", restorationContext.MappingVault, []byte("{\"ArchiveList\":[{\"ArchiveId\":\"mappingArchiveId\",\"Size\":42}]}"))
	mockStartRetrieveJob(glacierMock, restorationContext.MappingVault, "mappingArchiveId", "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	outputs := strings.Split(string(buffer.Bytes()), consts.LINE_BREAK)

	assert.True(t, regexp.MustCompile("Local mapping archive already exists with last modification date .+, retrieve a new mapping file \\?\\[y/N\\] Job to find mapping archive id has started \\(can last up to 4 hours\\): inventoryMappingJobId").MatchString(outputs[0]))
	assert.Equal(t, "Job has finished: inventoryMappingJobId", outputs[1])
	assert.Equal(t, "Job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId", outputs[2])
	assert.Equal(t, "Job has finished: retrieveMappingJobId", outputs[3])
	assert.Equal(t, "Mapping archive has been downloaded", outputs[4])
}

func assertMappingArchive(t *testing.T, expected string) {
	data, _ := ioutil.ReadFile("../../testtmp/cache/mapping.sqllite")
	assert.Equal(t, expected, string(data))
}

func assertCacheIsEmpty(t *testing.T) {
	assert.Equal(t, RegionVaultCache{}, ReadCache("../../testmp"))
}

type ReaderClosable struct {
	reader io.Reader
}

func (readerClosable ReaderClosable) Close() error {
	return nil
}

func (readerClosable ReaderClosable) Read(p []byte) (n int, err error) {
	return readerClosable.reader.Read(p)
}
