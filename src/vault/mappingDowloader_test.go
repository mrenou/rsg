package vault

import (
	"../awsutils"
	"../inputs"
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
)

func mockStartMappingJobInventory(glacierMock *GlacierMock, restorationContext *awsutils.RestorationContext) *mock.Call {
	params := &glacier.InitiateJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		VaultName: aws.String(restorationContext.MappingVault),
		JobParameters: &glacier.JobParameters{
			Description: aws.String("inventory " + restorationContext.MappingVault),
			Type:        aws.String("inventory-retrieval"),
			InventoryRetrievalParameters: &glacier.InventoryRetrievalJobInput{Limit: aws.String("2")},
		},
	}

	out := &glacier.InitiateJobOutput{
		JobId: aws.String("inventoryMappingJobId"),
	}

	return glacierMock.On("InitiateJob", params).Return(out, nil)
}

func mockDescribeJob(glacierMock *GlacierMock, restorationContext *awsutils.RestorationContext, jobId, vault string, completed bool) *mock.Call {
	params := &glacier.DescribeJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
	}

	out := &glacier.JobDescription{
		Completed: aws.Bool(completed),
	}

	return glacierMock.On("DescribeJob", params).Return(out, nil)
}

func mockDescribeJobErr(glacierMock *GlacierMock, restorationContext *awsutils.RestorationContext, jobId, vault string, err error) *mock.Call {
	params := &glacier.DescribeJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
	}

	return glacierMock.On("DescribeJob", params).Return(nil, err)
}

func mockOutputJob(glacierMock *GlacierMock, restorationContext *awsutils.RestorationContext, jobId, vault string, content []byte) *mock.Call {
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
	}

	out := &glacier.GetJobOutputOutput{
		Body:  newReaderClosable(bytes.NewReader(content)),
	}

	return glacierMock.On("GetJobOutput", params).Return(out, nil)
}

func mockStartRetrieveJob(glacierMock *GlacierMock, restorationContext *awsutils.RestorationContext, archiveId, description, retrievalByteRange, jobIdToReturn string) *mock.Call {
	params := &glacier.InitiateJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		VaultName: aws.String(restorationContext.MappingVault),
		JobParameters: &glacier.JobParameters{
			ArchiveId: aws.String(archiveId),
			//Description: aws.String(description),
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
	buffer := InitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)

	mockStartMappingJobInventory(glacierMock, restorationContext)
	mockDescribeJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, []byte("{\"ArchiveList\":[{\"ArchiveId\":\"mappingArchiveId\",\"Size\":42}]}"))
	mockStartRetrieveJob(glacierMock, restorationContext, "mappingArchiveId", "restore mapping from " + restorationContext.MappingVault, "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "job to find mapping archive id has started (can last up to 4 hours): inventoryMappingJobId\n" +
	"job has finished: inventoryMappingJobId\n" +
	"job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId\n" +
	"job has finished: retrieveMappingJobId\n" +
	"mapping archive has been downloaded\n", string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_inventory_job_in_progress(t *testing.T) {
	// Given
	buffer := InitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	restorationContext.RegionVaultCache = awsutils.RegionVaultCache{"inventoryMappingJobId", nil, ""}

	mockDescribeJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, false).Once()
	mockDescribeJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, []byte("{\"ArchiveList\":[{\"ArchiveId\":\"mappingArchiveId\",\"Size\":42}]}"))
	mockStartRetrieveJob(glacierMock, restorationContext, "mappingArchiveId", "restore mapping from " + restorationContext.MappingVault, "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	//Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "job to find mapping archive id is in progress (can last up to 4 hours): inventoryMappingJobId\n" +
	"job has finished: inventoryMappingJobId\n" +
	"job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId\n" +
	"job has finished: retrieveMappingJobId\n" +
	"mapping archive has been downloaded\n", string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_inventory_job_deprecated(t *testing.T) {
	// Given
	buffer := InitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	restorationContext.RegionVaultCache = awsutils.RegionVaultCache{"unknownInventoryMappingJobId", nil, ""}

	mockDescribeJobErr(glacierMock, restorationContext, "unknownInventoryMappingJobId", restorationContext.MappingVault, errors.New("The job ID was not found"))
	mockStartMappingJobInventory(glacierMock, restorationContext)
	mockDescribeJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, []byte("{\"ArchiveList\":[{\"ArchiveId\":\"mappingArchiveId\",\"Size\":42}]}"))
	mockStartRetrieveJob(glacierMock, restorationContext, "mappingArchiveId", "restore mapping from " + restorationContext.MappingVault, "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "WARNING: inventory job cahed for mapping vaul was not found\n" +
	"job to find mapping archive id has started (can last up to 4 hours): inventoryMappingJobId\n" +
	"job has finished: inventoryMappingJobId\n" +
	"job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId\n" +
	"job has finished: retrieveMappingJobId\n" +
	"mapping archive has been downloaded\n", string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_inventory_done(t *testing.T) {
	// Given
	buffer := InitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	restorationContext.RegionVaultCache = awsutils.RegionVaultCache{"inventoryMappingJobId", nil, ""}

	mockDescribeJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, []byte("{\"ArchiveList\":[{\"ArchiveId\":\"mappingArchiveId\",\"Size\":42}]}"))
	mockStartRetrieveJob(glacierMock, restorationContext, "mappingArchiveId", "restore mapping from " + restorationContext.MappingVault, "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId\n" +
	"job has finished: retrieveMappingJobId\n" +
	"mapping archive has been downloaded\n", string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_retrieve_job_in_progress(t *testing.T) {
	// Given
	buffer := InitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	restorationContext.RegionVaultCache = awsutils.RegionVaultCache{"", &awsutils.Archive{"mappingArchiveId", 42}, "retrieveMappingJobId"}

	mockDescribeJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, false).Once()
	mockDescribeJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	//Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "job to retrieve mapping archive is in progress (can last up to 4 hours): retrieveMappingJobId\n" +
	"job has finished: retrieveMappingJobId\n" +
	"mapping archive has been downloaded\n", string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_retrieve_job_deprecated(t *testing.T) {
	// Given
	buffer := InitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	restorationContext.RegionVaultCache = awsutils.RegionVaultCache{"", &awsutils.Archive{"mappingArchiveId", 42}, "unknownRetrieveMappingJobId"}

	mockDescribeJobErr(glacierMock, restorationContext, "unknownRetrieveMappingJobId", restorationContext.MappingVault, errors.New("The job ID was not found"))
	mockStartRetrieveJob(glacierMock, restorationContext, "mappingArchiveId", "restore mapping from " + restorationContext.MappingVault, "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "WARNING: retrieve mapping archive job cached was not found\n" +
	"job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId\n" +
	"job has finished: retrieveMappingJobId\n" +
	"mapping archive has been downloaded\n", string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_retrieve_done(t *testing.T) {
	// Given
	buffer := InitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	restorationContext.RegionVaultCache = awsutils.RegionVaultCache{"", &awsutils.Archive{"mappingArchiveId", 42}, "retrieveMappingJobId"}

	mockDescribeJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	assert.Equal(t, "mapping archive has been downloaded\n", string(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_mapping_already_exists(t *testing.T) {
	// Given
	buffer := InitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	restorationContext.RegionVaultCache = awsutils.RegionVaultCache{"", &awsutils.Archive{"mappingArchiveId", 42}, "retrieveMappingJobId"}

	ioutil.WriteFile("../../testtmp/cache/mapping.sqllite", []byte("hello !"), 0600)

	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("\n")))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	glacierMock.AssertNotCalled(t, mock.Anything)

	r := regexp.MustCompile("local mapping archive already exists with last modification date .+, retrieve a new mapping file \\?\\[y/N\\]")
	assert.True(t, r.Match(buffer.Bytes()))
}

func TestDownloadMappingArchive_download_mapping_with_mapping_already_exists_but_restart_download(t *testing.T) {
	// Given
	buffer := InitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)

	ioutil.WriteFile("../../testtmp/cache/mapping.sqllite", []byte("hello !"), 0600)

	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("y\n")))

	mockStartMappingJobInventory(glacierMock, restorationContext)
	mockDescribeJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "inventoryMappingJobId", restorationContext.MappingVault, []byte("{\"ArchiveList\":[{\"ArchiveId\":\"mappingArchiveId\",\"Size\":42}]}"))
	mockStartRetrieveJob(glacierMock, restorationContext, "mappingArchiveId", "restore mapping from " + restorationContext.MappingVault, "0-41", "retrieveMappingJobId")
	mockDescribeJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, true)
	mockOutputJob(glacierMock, restorationContext, "retrieveMappingJobId", restorationContext.MappingVault, []byte("hello !"))

	// When
	DownloadMappingArchive(restorationContext)

	// Then
	assertMappingArchive(t, "hello !")
	assertCacheIsEmpty(t)

	outputs := strings.Split(string(buffer.Bytes()), "\n")

	assert.True(t, regexp.MustCompile("local mapping archive already exists with last modification date .+, retrieve a new mapping file \\?\\[y/N\\]").MatchString(outputs[0]))
	assert.Equal(t, "job to find mapping archive id has started (can last up to 4 hours): inventoryMappingJobId", outputs[1])
	assert.Equal(t, "job has finished: inventoryMappingJobId", outputs[2])
	assert.Equal(t, "job to retrieve mapping archive has started (can last up to 4 hours): retrieveMappingJobId", outputs[3])
	assert.Equal(t, "job has finished: retrieveMappingJobId", outputs[4])
	assert.Equal(t, "mapping archive has been downloaded", outputs[5])
}

func assertMappingArchive(t *testing.T, expected string) {
	data, _ := ioutil.ReadFile("../../testtmp/cache/mapping.sqllite")
	assert.Equal(t, expected, string(data))
}

func assertCacheIsEmpty(t *testing.T) {
	assert.Equal(t, awsutils.RegionVaultCache{}, awsutils.ReadRegionVaultCache("region", "vault", "../../testmp"))
}

type ReaderClosable struct {
	reader io.Reader
}

func newReaderClosable(reader io.Reader) ReaderClosable {
	return ReaderClosable{reader}
}

func (readerClosable ReaderClosable) Close() error {
	return nil
}

func (readerClosable ReaderClosable) Read(p []byte) (n int, err error) {
	return readerClosable.reader.Read(p)
}
