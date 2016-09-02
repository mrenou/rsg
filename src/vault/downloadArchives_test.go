package vault

import (
	"testing"
	"../awsutils"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glacier"
	"bytes"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"strings"
	"../utils"
	"os"
	"../loggers"
)

func mockStartPartialRetrieveJob(glacierMock *GlacierMock, restorationContext *awsutils.RestorationContext, vault, archiveId, bytesRange, jobIdToReturn string) *mock.Call {
	var retrievalByteRange *string = nil
	if (bytesRange != "") {
		retrievalByteRange = aws.String(bytesRange)
	}
	params := &glacier.InitiateJobInput{
		AccountId: aws.String(restorationContext.AccountId),
		VaultName: aws.String(vault),
		JobParameters: &glacier.JobParameters{
			ArchiveId: aws.String(archiveId),
			//Description: aws.String(description),
			Type:        aws.String("archive-retrieval"),
			RetrievalByteRange: retrievalByteRange,
		},
	}

	out := &glacier.InitiateJobOutput{
		JobId: aws.String(jobIdToReturn),
	}
	return glacierMock.On("InitiateJob", params).Return(out, nil)
}

func mockPartialOutputJob(glacierMock *GlacierMock, restorationContext *awsutils.RestorationContext, jobId, vault, bytesRange string, content []byte) *mock.Call {
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String(restorationContext.AccountId),
		JobId:     aws.String(jobId),
		VaultName: aws.String(vault),
		Range: aws.String(bytesRange),
	}

	out := &glacier.GetJobOutputOutput{
		Body:  newReaderClosable(bytes.NewReader(content)),
	}

	return glacierMock.On("GetJobOutput", params).Return(out, nil)
}

func TestDownloadArchives_retrieve_and_download_file_in_one_part(t *testing.T) {
	// Given
	InitTest()
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	downloadContext := DownloadContext{
		restorationContext: restorationContext,
		bytesBySecond: 1,
		maxArchivesRetrievingSize: utils.S_1MB,
		downloadSpeedAutoUpdate: false,
		archivesRetrievingSize: 0,
		archivePartRetrieveListMaxSize: 1,
		archivePartRetrieveList: nil,
		hasRows: false,
		db: nil,
		rows:nil,
	}

	db, _ := sql.Open("sqlite3", restorationContext.GetMappingFilePath())
	db.Exec("CREATE TABLE `file_info_tb` (`key` INTEGER PRIMARY KEY AUTOINCREMENT, `basePath` TEXT,`archiveID` TEXT, fileSize INTEGER);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/file1.txt', 'archiveId1', 5);")
	db.Close()

	mockStartPartialRetrieveJob(glacierMock, restorationContext, restorationContext.Vault, "archiveId1", "0-4", "jobId1")
	mockDescribeJob(glacierMock, restorationContext, "jobId1", restorationContext.Vault, true)
	mockPartialOutputJob(glacierMock, restorationContext, "jobId1", restorationContext.Vault, "0-4", []byte("hello"))

	// When
	downloadContext.downloadArchives()

	// Then
	assertFileContent(t, "../../testtmp/dest/data/file1.txt", "hello")
}

func TestDownloadArchives_retrieve_and_download_file_with_multipart(t *testing.T) {
	// Given
	buffer := InitTest()
	loggers.InitLog(buffer, os.Stdout, os.Stdout, os.Stderr)
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	downloadContext := DownloadContext{
		restorationContext: restorationContext,
		bytesBySecond: 3496, // 1048800 on 5 min
		maxArchivesRetrievingSize: utils.S_1MB * 2,
		downloadSpeedAutoUpdate: false,
		archivesRetrievingSize: 0,
		archivePartRetrieveListMaxSize: 10,
		archivePartRetrieveList: nil,
		hasRows: false,
		db: nil,
		rows:nil,
	}

	db, _ := sql.Open("sqlite3", restorationContext.GetMappingFilePath())
	db.Exec("CREATE TABLE `file_info_tb` (`key` INTEGER PRIMARY KEY AUTOINCREMENT, `basePath` TEXT,`archiveID` TEXT, fileSize INTEGER);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/file1.txt', 'archiveId1', 4194304);")
	db.Close()

	mockStartPartialRetrieveJob(glacierMock, restorationContext, restorationContext.Vault, "archiveId1", "0-2097151", "jobId1").Once()
	mockDescribeJob(glacierMock, restorationContext, "jobId1", restorationContext.Vault, true).Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId1", restorationContext.Vault, "0-1048799", []byte(strings.Repeat("_", 1048800))).Once()

	mockStartPartialRetrieveJob(glacierMock, restorationContext, restorationContext.Vault, "archiveId1", "2097152-3145727", "jobId2").Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId1", restorationContext.Vault, "1048800-2097151", []byte(strings.Repeat("_", 1048352))).Once()
	mockDescribeJob(glacierMock, restorationContext, "jobId2", restorationContext.Vault, true).Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId2", restorationContext.Vault, "0-447", []byte(strings.Repeat("_", 448))).Once()

	mockStartPartialRetrieveJob(glacierMock, restorationContext, restorationContext.Vault, "archiveId1", "3145728-4194303", "jobId3").Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId2", restorationContext.Vault, "448-1048575", []byte(strings.Repeat("_", 1048128))).Once()
	mockDescribeJob(glacierMock, restorationContext, "jobId3", restorationContext.Vault, true).Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId3", restorationContext.Vault, "0-671", []byte(strings.Repeat("_", 672)))

	mockPartialOutputJob(glacierMock, restorationContext, "jobId3", restorationContext.Vault, "672-1048575", append([]byte(strings.Repeat("_", 1047899)), []byte("hello")...))

	// When
	downloadContext.downloadArchives()

	// Then
	assertFileContent(t, "../../testtmp/dest/data/file1.txt", strings.Repeat("_", 4194299) + "hello")
}

func assertFileContent(t *testing.T, filePath, expected string) {
	data, _ := ioutil.ReadFile(filePath)
	assert.Equal(t, expected, string(data))
}