package vault

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/service/glacier/glacieriface"
	"strings"
	"github.com/aws/aws-sdk-go/aws/session"
)

type SynologyCoupleVault struct {
	region       string
	name         string
	dataVault    *glacier.DescribeVaultOutput
	mappingVault *glacier.DescribeVaultOutput
}

func GetSynologyVaultsOnAllRegions(accountId string, sessionValue *session.Session) []*SynologyCoupleVault {
	//regions := []string{"us-east-1", "us-west-1", "us-west-2", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-northeast-2", "ap-southeast-1", "ap-southeast-2", "sa-east-1"}
	regions := []string{"us-east-1", "us-west-1", "us-west-2", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-northeast-2", "ap-southeast-2"}
	synologyCoupleVaults := []*SynologyCoupleVault{}
	for _, region := range regions {
		fmt.Printf("Search synology backups for %s...", region)
		glacierClient := glacier.New(sessionValue, &aws.Config{Region: aws.String(region)})
		synologyCoupleVaultsForRegion := GetSynologyVaultsForRegion(accountId, glacierClient, region)
		fmt.Printf(" %d found\n", len(synologyCoupleVaultsForRegion))
		synologyCoupleVaults = append(synologyCoupleVaults, synologyCoupleVaultsForRegion...)
	}
	return synologyCoupleVaults;

}

func GetSynologyVaultsForRegion(accountId string, glacierClient glacieriface.GlacierAPI, region string) []*SynologyCoupleVault {
	haveResults := true

	possibleDataVaults := map[string]*glacier.DescribeVaultOutput{}
	synologyCoupleVaults := []*SynologyCoupleVault{}
	resp := &glacier.ListVaultsOutput{}
	var err error
	for haveResults {
		resp, err = getVaults(accountId, glacierClient, resp.Marker)

		if err != nil {
			fmt.Println(err.Error())
			return nil
		}

		for _, vault := range resp.VaultList {
			vaultName := *vault.VaultName
			if strings.HasSuffix(vaultName, "_mapping") {
				dataVaultName := vaultName[0:strings.LastIndex(vaultName, "_mapping")]
				if dataVault, ok := possibleDataVaults[dataVaultName]; ok {
					synologyCoupleVaults = append(synologyCoupleVaults, &SynologyCoupleVault{region, dataVaultName, dataVault, vault})
				}
			} else {
				possibleDataVaults[*vault.VaultName] = vault
			}
		}

		haveResults = resp.Marker != nil
	}
	return synologyCoupleVaults
}

func getVaults(accountId string, glacierClient glacieriface.GlacierAPI, marker *string) (*glacier.ListVaultsOutput, error) {
	params := &glacier.ListVaultsInput{
		AccountId: aws.String(accountId),
		Marker:    marker,
	}
	return glacierClient.ListVaults(params)
}

