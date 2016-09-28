package core

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/service/glacier/glacieriface"
	"strings"
	"errors"
	"rsg/loggers"
	"rsg/utils"
	"rsg/awsutils"
)

var Regions = []string{"us-east-1", "us-west-1", "us-west-2", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-northeast-2", "ap-southeast-2"}

type SynologyCoupleVault struct {
	Region       string
	Name         string
	DataVault    *glacier.DescribeVaultOutput
	MappingVault *glacier.DescribeVaultOutput
}

func GetSynologyVaults(regionFilter, vaultFilter string) ([]*SynologyCoupleVault, error) {
	loggers.Printfln(loggers.OptionalInfo, "Scan synology backup vaults...")
	if regionFilter != "" {
		return getSynologyVaultsOnOneRegions(regionFilter, vaultFilter)
	}
	return getSynologyVaultsOnAllRegions(vaultFilter)
}

func getSynologyVaultsOnOneRegions(regionFilter, vaultFilter string) ([]*SynologyCoupleVault, error) {
	if !utils.Contains(Regions, regionFilter) {
		return nil, errors.New(fmt.Sprintf("Region %s is not allowed, use : %s", regionFilter, strings.Join(Regions, ", ")))
	}
	glacierClient := glacier.New(awsutils.Session, &aws.Config{Region: aws.String(regionFilter)})
	return getSynologyVaultsForRegion(glacierClient, regionFilter, vaultFilter)
}

func getSynologyVaultsOnAllRegions(vaultFilter string) ([]*SynologyCoupleVault, error) {
	synologyCoupleVaults := []*SynologyCoupleVault{}
	for _, region := range Regions {

		glacierClient := glacier.New(awsutils.Session, &aws.Config{Region: aws.String(region)})
		synologyCoupleVaultsForRegion, err := getSynologyVaultsForRegion(glacierClient, region, vaultFilter)
		if err != nil {
			return nil, err
		}
		synologyCoupleVaults = append(synologyCoupleVaults, synologyCoupleVaultsForRegion...)
	}
	return synologyCoupleVaults, nil;
}

func getSynologyVaultsForRegion(glacierClient glacieriface.GlacierAPI, region string, vaultFilter string) ([]*SynologyCoupleVault, error) {
	loggers.Printfln(loggers.Verbose, "Get vaults for region %s with vaultFilter=%s", region, vaultFilter)
	haveResults := true
	possibleDataVaults := map[string]*glacier.DescribeVaultOutput{}
	synologyCoupleVaults := []*SynologyCoupleVault{}
	resp := &glacier.ListVaultsOutput{}
	var err error
	for haveResults {
		resp, err = awsutils.GetVaults(glacierClient, resp.Marker)
		if err != nil {
			return nil, err
		}
		for _, vault := range resp.VaultList {
			vaultName := *vault.VaultName
			if strings.HasSuffix(vaultName, "_mapping") {
				dataVaultName := vaultName[0:strings.LastIndex(vaultName, "_mapping")]
				if dataVault, ok := possibleDataVaults[dataVaultName]; ok {
					if vaultFilter != "" && vaultFilter == dataVaultName {
						loggers.Printfln(loggers.Verbose, "Vault found %s", dataVaultName)
						return []*SynologyCoupleVault{{Region: region,
							Name: dataVaultName,
							DataVault: dataVault,
							MappingVault: vault}}, nil
					} else {
						loggers.Printfln(loggers.Verbose, "Vault added %s", dataVaultName)
						synologyCoupleVaults = append(synologyCoupleVaults, &SynologyCoupleVault{Region: region,
							Name: dataVaultName,
							DataVault: dataVault,
							MappingVault: vault})
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



