package vault

import (
	"strings"
	"github.com/aws/aws-sdk-go/aws/session"
	"../loggers"
	"../utils"
	"../inputs"
)

func SelectRegionVault(accountId string, sessionValue *session.Session, givenRegion, givenVault string) (string, string) {
	synologyCoupleVaults, err := GetSynologyVaults(accountId, sessionValue, givenRegion, givenVault)
	utils.ExitIfError(err)
	return selectRegionVaultFromSynologyVaults(synologyCoupleVaults)
}

func selectRegionVaultFromSynologyVaults(synologyCoupleVaults []*SynologyCoupleVault) (string, string) {
	var synologyCoupleVaultToUse *SynologyCoupleVault
	switch len(synologyCoupleVaults) {
	case 0:
	case 1:
		synologyCoupleVaultToUse = synologyCoupleVaults[0]
	default:
		for _, synologyCoupleVault := range synologyCoupleVaults {
			loggers.Info.Printf("%s:%s\n", synologyCoupleVault.Region, synologyCoupleVault.Name)
		}
		for synologyCoupleVaultToUse == nil {
			region := readRegionFromStdIn()
			vault := readVaultFromStdIn()
			synologyCoupleVaultToUse = getVaultIfExist(region, vault, synologyCoupleVaults)
			if synologyCoupleVaultToUse == nil {
				loggers.Info.Print("vault or region doesn't exist. Try again...")
			}
		}
	}

	if synologyCoupleVaultToUse != nil {
		loggers.Info.Printf("synology backup vault used for the restoration: %s:%s", synologyCoupleVaultToUse.Region, synologyCoupleVaultToUse.Name)
		return synologyCoupleVaultToUse.Region, synologyCoupleVaultToUse.Name
	} else {
		loggers.Error.Print("no synology backup vault found")
		return "", ""
	}

}

func readRegionFromStdIn() string {
	loggers.Info.Print("Select the region of the vault to use for the restoration:")
	region, err := inputs.StdinReader.ReadString('\n')
	utils.ExitIfError(err)
	return strings.TrimSuffix(region, "\n")
}

func readVaultFromStdIn() string {
	loggers.Info.Print("Select the vault to use for the restoration:")
	vault, err := inputs.StdinReader.ReadString('\n')
	utils.ExitIfError(err)
	return strings.TrimSuffix(vault, "\n")
}

func getVaultIfExist(region, vault string, synologyCoupleVaults []*SynologyCoupleVault) *SynologyCoupleVault {
	for _, synologyCoupleVault := range synologyCoupleVaults {
		if synologyCoupleVault.Region == region && synologyCoupleVault.Name == vault {
			return synologyCoupleVault
		}
	}
	return nil
}

