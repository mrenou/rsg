package core

import (
	"bytes"
	"os"
	"time"
	"rsg/loggers"
	"rsg/awsutils"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/mock"
	"github.com/aws/aws-sdk-go/service/glacier"
	"io/ioutil"
	"io"
	"path/filepath"
	"rsg/utils"
)

func CommonInitTest() *bytes.Buffer {
	loggers.VerboseFlag = true
	loggers.InitDefaultLog()
	buffer := new(bytes.Buffer)
	os.RemoveAll("../../testtmp")
	os.MkdirAll("../../testtmp/cache", 0700)
	loggers.InitLog(os.Stdout, buffer, buffer, buffer, os.Stderr)
	awsutils.WaitTime = 1 * time.Nanosecond
	awsutils.AccountId = "accountId"
	awsutils.JobIdsAtStartup.MappingInventoryJobId = ""
	awsutils.JobIdsAtStartup.MappingRetrievalJobId = ""
	return buffer
}

func InitTestWithGlacier() (*GlacierMock, *RestorationContext) {
	glacierMock := new(GlacierMock)
	restorationContext := DefaultRestorationContext(glacierMock)
	err := os.MkdirAll(filepath.Dir(restorationContext.DestinationDirPath + "/"), 0700)
	utils.ExitIfError(err)
	mockGetDataRetrievalPolicy(glacierMock, "accountId", "FreeTier")
	return glacierMock, restorationContext
}

func DefaultRestorationContext(glacierMock *GlacierMock) *RestorationContext {
	return &RestorationContext{GlacierClient: glacierMock,
		WorkingDirPath: "../../testtmp/cache",
		Region: "region",
		Vault: "vault",
		MappingVault: "vault_mapping",
		RegionVaultCache: RegionVaultCache{},
		DestinationDirPath: "../../testtmp/dest",
		BytesBySecond: 0,
		Options: RestorationOptions{}}
}

type SessionMock struct {
	session.Session
	mock.Mock
}

type GlacierMock struct {
	glacier.Glacier
	mock.Mock
}

func (m *GlacierMock) ListVaults(input *glacier.ListVaultsInput) (*glacier.ListVaultsOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*glacier.ListVaultsOutput), args.Error(1)
}

func (m *GlacierMock) InitiateJob(input *glacier.InitiateJobInput) (*glacier.InitiateJobOutput, error) {
	args := m.Called(input)
	if args.Get(0) != nil {
		return args.Get(0).(*glacier.InitiateJobOutput), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *GlacierMock) DescribeJob(input *glacier.DescribeJobInput) (*glacier.JobDescription, error) {
	args := m.Called(input)
	if args.Get(0) != nil {
		return args.Get(0).(*glacier.JobDescription), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *GlacierMock) GetJobOutput(input *glacier.GetJobOutputInput) (*glacier.GetJobOutputOutput, error) {
	args := m.Called(input)
	getJobOutputOutput := args.Get(0).(*glacier.GetJobOutputOutput)
	content, _ := ioutil.ReadAll(getJobOutputOutput.Body)
	getJobOutputOutput.Body = newReaderClosable(bytes.NewReader(content))
	getJobOutputOutputCopy := &glacier.GetJobOutputOutput{
		AcceptRanges: getJobOutputOutput.AcceptRanges,
		ArchiveDescription: getJobOutputOutput.ArchiveDescription,
		Body: newReaderClosable(bytes.NewReader(content)),
		Checksum: getJobOutputOutput.Checksum,
		ContentRange: getJobOutputOutput.ContentRange,
		ContentType: getJobOutputOutput.ContentType,
		Status: getJobOutputOutput.Status}
	return getJobOutputOutputCopy, args.Error(1)
}

func (m *GlacierMock) GetDataRetrievalPolicy(input *glacier.GetDataRetrievalPolicyInput) (*glacier.GetDataRetrievalPolicyOutput, error) {
	args := m.Called(input)
	if args.Get(0) != nil {
		return args.Get(0).(*glacier.GetDataRetrievalPolicyOutput), args.Error(1)
	}
	return nil, args.Error(1)
}

func newReaderClosable(reader io.Reader) ReaderClosable {
	return ReaderClosable{reader}
}

func mockGetDataRetrievalPolicy(glacierMock *GlacierMock, accountId, strategy string) *mock.Call {
	input := &glacier.GetDataRetrievalPolicyInput{
		AccountId:  &accountId,
	}
	out := &glacier.GetDataRetrievalPolicyOutput{
		Policy: &glacier.DataRetrievalPolicy{Rules: []*glacier.DataRetrievalRule{&glacier.DataRetrievalRule{Strategy: &strategy}}},
	}

	return glacierMock.On("GetDataRetrievalPolicy", input).Return(out, nil)
}


