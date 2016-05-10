package vault

import (
	"github.com/stretchr/testify/mock"
	"testing"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"github.com/aws/aws-sdk-go/aws/session"
	"../loggers"
)

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

func TestGetSynologyVaults_should_return_one_vault(t *testing.T) {
	loggers.InitDefaultLog()
	// Given

	glacierMock := new(GlacierMock)

	params := &glacier.ListVaultsInput{
		AccountId: aws.String("42"), // Required
		Limit:     nil,
		Marker:    nil,
	}

	out := &glacier.ListVaultsOutput{VaultList: []*glacier.DescribeVaultOutput{
		&glacier.DescribeVaultOutput{VaultName: aws.String("a")},
		&glacier.DescribeVaultOutput{VaultName: aws.String("vault1")},
		&glacier.DescribeVaultOutput{VaultName: aws.String("vault1_mapping")},
		&glacier.DescribeVaultOutput{VaultName: aws.String("z")},
	}}

	glacierMock.On("ListVaults", params).Return(out, nil)

	// When

	synoVaultCouples, _ := getSynologyVaultsForRegion("42", glacierMock, "region", "")

	// Then

	assert.Len(t, synoVaultCouples, 1)
	assert.Equal(t, "region", synoVaultCouples[0].Region)
	assert.Equal(t, "vault1", synoVaultCouples[0].Name)
	assert.Equal(t, "vault1", *synoVaultCouples[0].DataVault.VaultName)
	assert.Equal(t, "vault1_mapping", *synoVaultCouples[0].MappingVault.VaultName)
}

func TestGetSynologyVaults_should_return_two_vault2_with_two_aws_requests(t *testing.T) {
	loggers.InitDefaultLog()
	// Given
	glacierMock := new(GlacierMock)

	params := &glacier.ListVaultsInput{
		AccountId: aws.String("42"), // Required
		Limit:     nil,
		Marker:    nil,
	}

	out := &glacier.ListVaultsOutput{Marker: aws.String("marker"), VaultList: []*glacier.DescribeVaultOutput{
		&glacier.DescribeVaultOutput{VaultName: aws.String("a")},
		&glacier.DescribeVaultOutput{VaultName: aws.String("vault1")},
		&glacier.DescribeVaultOutput{VaultName: aws.String("vault1_mapping")},
		&glacier.DescribeVaultOutput{VaultName: aws.String("z")},
		&glacier.DescribeVaultOutput{VaultName: aws.String("vault2")},
	}}

	glacierMock.On("ListVaults", params).Return(out, nil)

	params = &glacier.ListVaultsInput{
		AccountId: aws.String("42"), // Required
		Limit:     nil,
		Marker:    aws.String("marker"),
	}

	out = &glacier.ListVaultsOutput{VaultList: []*glacier.DescribeVaultOutput{
		&glacier.DescribeVaultOutput{VaultName: aws.String("b")},
		&glacier.DescribeVaultOutput{VaultName: aws.String("vault2_mapping")},
		&glacier.DescribeVaultOutput{VaultName: aws.String("y")},
	}}

	glacierMock.On("ListVaults", params).Return(out, nil)

	// When

	synoVaultCouples, _ := getSynologyVaultsForRegion("42", glacierMock, "region", "")

	// Then

	assert.Len(t, synoVaultCouples, 2)
	assert.Equal(t, "region", synoVaultCouples[0].Region)
	assert.Equal(t, "vault1", synoVaultCouples[0].Name)
	assert.Equal(t, "vault1", *synoVaultCouples[0].DataVault.VaultName)
	assert.Equal(t, "vault1_mapping", *synoVaultCouples[0].MappingVault.VaultName)

	assert.Equal(t, "region", synoVaultCouples[1].Region)
	assert.Equal(t, "vault2", synoVaultCouples[1].Name)
	assert.Equal(t, "vault2", *synoVaultCouples[1].DataVault.VaultName)
	assert.Equal(t, "vault2_mapping", *synoVaultCouples[1].MappingVault.VaultName)
}



