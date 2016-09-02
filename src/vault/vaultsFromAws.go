package vault

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/service/glacier/glacieriface"
	"strings"
	"github.com/aws/aws-sdk-go/aws/session"
	"errors"
	"../loggers"
	"../utils"
)

var Regions = []string{"us-east-1", "us-west-1", "us-west-2", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-northeast-2", "ap-southeast-2"}

type SynologyCoupleVault struct {
	Region       string
	Name         string
	DataVault    *glacier.DescribeVaultOutput
	MappingVault *glacier.DescribeVaultOutput
}

func GetSynologyVaults(accountId string, sessionValue *session.Session, regionFilter, vaultFilter string) ([]*SynologyCoupleVault, error) {
	if regionFilter != "" {
		return getSynologyVaultsOnOneRegions(accountId, sessionValue, regionFilter, vaultFilter)
	}
	return getSynologyVaultsOnAllRegions(accountId, sessionValue, regionFilter, vaultFilter)
}

func getSynologyVaultsOnOneRegions(accountId string, sessionValue *session.Session, regionFilter, vaultFilter string) ([]*SynologyCoupleVault, error) {
	if !utils.Contains(Regions, regionFilter) {
		return nil, errors.New(fmt.Sprintf("region %s is not allowed, use : %s", regionFilter, strings.Join(Regions, ", ")))
	}
	glacierClient := glacier.New(sessionValue, &aws.Config{Region: aws.String(regionFilter)})
	return getSynologyVaultsForRegion(accountId, glacierClient, regionFilter, vaultFilter)
}

func getSynologyVaultsOnAllRegions(accountId string, sessionValue *session.Session, regionFilter, vaultFilter string) ([]*SynologyCoupleVault, error) {
	synologyCoupleVaults := []*SynologyCoupleVault{}
	for _, region := range Regions {

		glacierClient := glacier.New(sessionValue, &aws.Config{Region: aws.String(region)})
		synologyCoupleVaultsForRegion, err := getSynologyVaultsForRegion(accountId, glacierClient, region, vaultFilter)
		if err != nil {
			return nil, err
		}
		synologyCoupleVaults= append(synologyCoupleVaults, synologyCoupleVaultsForRegion...)
	}
	return synologyCoupleVaults, nil;
}

func getSynologyVaultsForRegion(accountId string, glacierClient glacieriface.GlacierAPI, region string, vaultFilter string) ([]*SynologyCoupleVault, error) {
	loggers.Printf(loggers.Debug, "get vault for region %s with vaultFilter=%s", region, vaultFilter)
	haveResults := true
	possibleDataVaults := map[string]*glacier.DescribeVaultOutput{}
	synologyCoupleVaults := []*SynologyCoupleVault{}
	resp := &glacier.ListVaultsOutput{}
	var err error
	for haveResults {
		resp, err = getVaults(accountId, glacierClient, resp.Marker)
		if err != nil {
			return nil, err
		}
		for _, vault := range resp.VaultList {
			vaultName := *vault.VaultName
			if strings.HasSuffix(vaultName, "_mapping") {
				dataVaultName := vaultName[0:strings.LastIndex(vaultName, "_mapping")]
				if dataVault, ok := possibleDataVaults[dataVaultName]; ok {
					if vaultFilter != "" && vaultFilter == dataVaultName {
						loggers.Printf(loggers.Debug, "vault found %s", dataVaultName)
						return []*SynologyCoupleVault{&SynologyCoupleVault{region, dataVaultName, dataVault, vault}}, nil
					} else {
						loggers.Printf(loggers.Debug, "vault added %s", dataVaultName)
						synologyCoupleVaults = append(synologyCoupleVaults, &SynologyCoupleVault{region, dataVaultName, dataVault, vault})
					}
				}
			} else {
				possibleDataVaults[*vault.VaultName] = vault
			}
		}

		haveResults = resp.Marker != nil
	}
	return synologyCoupleVaults, nil
}

func getVaults(accountId string, glacierClient glacieriface.GlacierAPI, marker *string) (*glacier.ListVaultsOutput, error) {
	params := &glacier.ListVaultsInput{
		AccountId: aws.String(accountId),
		Marker:    marker,
	}
	return glacierClient.ListVaults(params)
}

