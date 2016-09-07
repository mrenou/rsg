package vault

import (
	"testing"
	"rsg/awsutils"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glacier"
	"bytes"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"strings"
	"rsg/utils"
	"os"
	"path/filepath"
	"errors"
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

func mockStartPartialRetrieveJobForAny(glacierMock *GlacierMock, jobIdToReturn string) *mock.Call {
	out := &glacier.InitiateJobOutput{
		JobId: aws.String(jobIdToReturn),
	}
	return glacierMock.On("InitiateJob", mock.AnythingOfType("*glacier.InitiateJobInput")).Return(out, nil)
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

func mockPartialOutputJobForAny(glacierMock *GlacierMock, content []byte) *mock.Call {
	out := &glacier.GetJobOutputOutput{
		Body:  newReaderClosable(bytes.NewReader(content)),
	}

	return glacierMock.On("GetJobOutput", mock.AnythingOfType("*glacier.GetJobOutputInput")).Return(out, nil)
}

func mockDescribeJobForAny(glacierMock *GlacierMock, completed bool) *mock.Call {
	out := &glacier.JobDescription{
		Completed: aws.Bool(completed),
	}

	return glacierMock.On("DescribeJob", mock.AnythingOfType("*glacier.DescribeJobInput")).Return(out, nil)
}



func InitTest() (*GlacierMock, *awsutils.RestorationContext) {
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	err := os.MkdirAll(filepath.Dir(restorationContext.DestinationDirPath + "/"), 0700)
	utils.ExitIfError(err)
	return glacierMock, restorationContext
}

func TestDownloadArchives_retrieve_and_download_file_in_one_part(t *testing.T) {
	// Given
	CommonInitTest()
	glacierMock, restorationContext := InitTest()
	downloadContext := DownloadContext{
		restorationContext: restorationContext,
		bytesBySecond: 1,
		maxArchivesRetrievingSize: utils.S_1MB,
		downloadSpeedAutoUpdate: false,
		archivesRetrievingSize: 0,
		archivePartRetrieveListMaxSize: 1,
		archivePartRetrieveList: nil,
		hasArchiveRows: false,
		db: nil,
		archiveRows:nil,
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
	CommonInitTest()
	glacierMock, restorationContext := InitTest()
	downloadContext := DownloadContext{
		restorationContext: restorationContext,
		bytesBySecond: 3496, // 1048800 on 5 min
		maxArchivesRetrievingSize: utils.S_1MB * 2,
		downloadSpeedAutoUpdate: false,
		archivesRetrievingSize: 0,
		archivePartRetrieveListMaxSize: 10,
		archivePartRetrieveList: nil,
		hasArchiveRows: false,
		db: nil,
		archiveRows:nil,
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

func TestDownloadArchives_retrieve_and_download_2_files_with_multipart(t *testing.T) {
	// Given
	CommonInitTest()
	glacierMock, restorationContext := InitTest()
	downloadContext := DownloadContext{
		restorationContext: restorationContext,
		bytesBySecond: 3496, // 1048800 on 5 min
		maxArchivesRetrievingSize: utils.S_1MB * 2,
		downloadSpeedAutoUpdate: false,
		archivesRetrievingSize: 0,
		archivePartRetrieveListMaxSize: 10,
		archivePartRetrieveList: nil,
		hasArchiveRows: false,
		db: nil,
		archiveRows:nil,
	}

	db, _ := sql.Open("sqlite3", restorationContext.GetMappingFilePath())
	db.Exec("CREATE TABLE `file_info_tb` (`key` INTEGER PRIMARY KEY AUTOINCREMENT, `basePath` TEXT,`archiveID` TEXT, fileSize INTEGER);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/file1.txt', 'archiveId1', 4194304);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/file2.txt', 'archiveId2', 2097152);")
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

	mockStartPartialRetrieveJob(glacierMock, restorationContext, restorationContext.Vault, "archiveId2", "0-1048575", "jobId4").Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId3", restorationContext.Vault, "672-1048575", append([]byte(strings.Repeat("_", 1047899)), []byte("hello")...))
	mockDescribeJob(glacierMock, restorationContext, "jobId4", restorationContext.Vault, true).Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId4", restorationContext.Vault, "0-895", []byte(strings.Repeat("_", 896)))

	mockStartPartialRetrieveJob(glacierMock, restorationContext, restorationContext.Vault, "archiveId2", "1048576-2097151", "jobId5").Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId4", restorationContext.Vault, "896-1048575", []byte(strings.Repeat("_", 1047680)))
	mockDescribeJob(glacierMock, restorationContext, "jobId5", restorationContext.Vault, true).Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId5", restorationContext.Vault, "0-1119", []byte(strings.Repeat("_", 1120)))

	mockPartialOutputJob(glacierMock, restorationContext, "jobId5", restorationContext.Vault, "1120-1048575", append([]byte(strings.Repeat("_", 1047451)), []byte("olleh")...))

	// When
	downloadContext.downloadArchives()

	// Then
	assertFileContent(t, "../../testtmp/dest/data/file1.txt", strings.Repeat("_", 4194299) + "hello")
	assertFileContent(t, "../../testtmp/dest/data/file2.txt", strings.Repeat("_", 2097147) + "olleh")
}

func TestDownloadArchives_retrieve_and_download_3_files_with_2_identical(t *testing.T) {
	// Given
	CommonInitTest()
	glacierMock, restorationContext := InitTest()
	downloadContext := DownloadContext{
		restorationContext: restorationContext,
		bytesBySecond: 3496, // 1048800 on 5 min
		maxArchivesRetrievingSize: utils.S_1MB * 2,
		downloadSpeedAutoUpdate: false,
		archivesRetrievingSize: 0,
		archivePartRetrieveListMaxSize: 10,
		archivePartRetrieveList: nil,
		hasArchiveRows: false,
		db: nil,
		archiveRows:nil,
	}

	db, _ := sql.Open("sqlite3", restorationContext.GetMappingFilePath())
	db.Exec("CREATE TABLE `file_info_tb` (`key` INTEGER PRIMARY KEY AUTOINCREMENT, `basePath` TEXT,`archiveID` TEXT, fileSize INTEGER);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/file1.txt', 'archiveId1', 4194304);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/file2.txt', 'archiveId2', 2097152);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/file3.txt', 'archiveId1', 4194304);")
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

	mockStartPartialRetrieveJob(glacierMock, restorationContext, restorationContext.Vault, "archiveId2", "0-1048575", "jobId4").Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId3", restorationContext.Vault, "672-1048575", append([]byte(strings.Repeat("_", 1047899)), []byte("hello")...))
	mockDescribeJob(glacierMock, restorationContext, "jobId4", restorationContext.Vault, true).Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId4", restorationContext.Vault, "0-895", []byte(strings.Repeat("_", 896)))

	mockStartPartialRetrieveJob(glacierMock, restorationContext, restorationContext.Vault, "archiveId2", "1048576-2097151", "jobId5").Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId4", restorationContext.Vault, "896-1048575", []byte(strings.Repeat("_", 1047680)))
	mockDescribeJob(glacierMock, restorationContext, "jobId5", restorationContext.Vault, true).Once()
	mockPartialOutputJob(glacierMock, restorationContext, "jobId5", restorationContext.Vault, "0-1119", []byte(strings.Repeat("_", 1120)))

	mockPartialOutputJob(glacierMock, restorationContext, "jobId5", restorationContext.Vault, "1120-1048575", append([]byte(strings.Repeat("_", 1047451)), []byte("olleh")...))

	// When
	downloadContext.downloadArchives()

	// Then
	assertFileContent(t, "../../testtmp/dest/data/file1.txt", strings.Repeat("_", 4194299) + "hello")
	assertFileContent(t, "../../testtmp/dest/data/file2.txt", strings.Repeat("_", 2097147) + "olleh")
	assertFileContent(t, "../../testtmp/dest/data/file3.txt", strings.Repeat("_", 4194299) + "hello")
}

func TestDownloadArchives_retrieve_and_download_only_filtered_files(t *testing.T) {
	// Given
	CommonInitTest()
	glacierMock, restorationContext := InitTest()
	restorationContext.Filters = []string{"data/folder/*", "*.info", "data/file??.bin", "data/iwantthis" }
	downloadContext := DownloadContext{
		restorationContext: restorationContext,
		bytesBySecond: 3496, // 1048800 on 5 min
		maxArchivesRetrievingSize: utils.S_1MB * 2,
		downloadSpeedAutoUpdate: false,
		archivesRetrievingSize: 0,
		archivePartRetrieveListMaxSize: 10,
		archivePartRetrieveList: nil,
		hasArchiveRows: false,
		db: nil,
		archiveRows:nil,
	}

	db, _ := sql.Open("sqlite3", restorationContext.GetMappingFilePath())
	db.Exec("CREATE TABLE `file_info_tb` (`key` INTEGER PRIMARY KEY AUTOINCREMENT, `basePath` TEXT,`archiveID` TEXT, fileSize INTEGER);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/folder/file1.txt', 'archiveId1', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/folder/file2.bin', 'archiveId2', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/folderno/no.bin', 'archiveId3', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/no', 'archiveId4', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/otherfolder/no', 'archiveId5', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/otherfolder/file3.info', 'archiveId6', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/otherfolder/no.txt', 'archiveId7', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/file4.info', 'archiveId8', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/file41.bin', 'archiveId9', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/file42.bin', 'archiveId10', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/filenop.bin', 'archiveId11', 2);")
	db.Exec("INSERT INTO `file_info_tb` (basePath, archiveID, fileSize) VALUES ('data/iwantthis', 'archiveId12', 2);")
	db.Close()

	mockStartPartialRetrieveJobForAny(glacierMock, "jobId")
	mockDescribeJobForAny(glacierMock, true)
	mockPartialOutputJobForAny(glacierMock, []byte("ok"))

	// When
	downloadContext.downloadArchives()

	// Then
	assertFileContent(t, "../../testtmp/dest/data/folder/file1.txt", "ok")
	assertFileContent(t, "../../testtmp/dest/data/folder/file2.bin", "ok")
	assertFileDoestntExist(t, "../../testtmp/dest/data/folder/no.bin")
	assertFileDoestntExist(t, "../../testtmp/dest/data/no")
	assertFileDoestntExist(t, "../../testtmp/dest/data/otherfolder/no")
	assertFileContent(t, "../../testtmp/dest/data/otherfolder/file3.info", "ok")
	assertFileDoestntExist(t, "../../testtmp/dest/data/otherfolder/no.txt")
	assertFileContent(t, "../../testtmp/dest/data/file4.info", "ok")
	assertFileContent(t, "../../testtmp/dest/data/file41.bin", "ok")
	assertFileContent(t, "../../testtmp/dest/data/file42.bin", "ok")
	assertFileDoestntExist(t, "../../testtmp/dest/data/filenop.bin")
	assertFileContent(t, "../../testtmp/dest/data/iwantthis", "ok")
}

func assertFileContent(t *testing.T, filePath, expected string) {
	data, _ := ioutil.ReadFile(filePath)
	assert.Equal(t, expected, string(data))
}

func assertFileDoestntExist(t *testing.T, filePath string) {
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		assert.Fail(t, "path should not exist")
	}
}