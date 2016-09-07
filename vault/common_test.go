package vault

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
)

func CommonInitTest() *bytes.Buffer {
	loggers.DebugFlag = true
	loggers.InitDefaultLog()
	buffer := new(bytes.Buffer)
	os.RemoveAll("../../testtmp")
	os.MkdirAll("../../testtmp/cache", 0700)
	loggers.InitLog(os.Stdout, buffer, buffer, os.Stderr)
	awsutils.WaitTime = 1 * time.Nanosecond
	return buffer
}

func DefaultRestorationContext(glacierMock *GlacierMock) *awsutils.RestorationContext {
	return &awsutils.RestorationContext{glacierMock, "../../testtmp/cache", "region", "vault", "vault_mapping", "acountId", awsutils.RegionVaultCache{}, "../../testtmp/dest", []string{}, 0}
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

func newReaderClosable(reader io.Reader) ReaderClosable {
	return ReaderClosable{reader}
}


