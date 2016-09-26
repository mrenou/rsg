package core

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"rsg/loggers"
	"rsg/utils"
	"rsg/inputs"
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
			loggers.Printfln(loggers.Info, "%s:%s", synologyCoupleVault.Region, synologyCoupleVault.Name)
		}
		for synologyCoupleVaultToUse == nil {
			region := inputs.QueryString("Select the region of the vault to use:")
			vault := inputs.QueryString("Select the vault to use:")
			synologyCoupleVaultToUse = getVaultIfExist(region, vault, synologyCoupleVaults)
			if synologyCoupleVaultToUse == nil {
				loggers.Println(loggers.Info, "Vault or region doesn't exist. Try again...")
			}
		}
	}

	if synologyCoupleVaultToUse != nil {
		loggers.Printfln(loggers.OptionalInfo, "Synology backup vault used: %s:%s", synologyCoupleVaultToUse.Region, synologyCoupleVaultToUse.Name)
		return synologyCoupleVaultToUse.Region, synologyCoupleVaultToUse.Name
	} else {
		loggers.Println(loggers.Error, "No synology backup vault found")
		return "", ""
	}

}

func getVaultIfExist(region, vault string, synologyCoupleVaults []*SynologyCoupleVault) *SynologyCoupleVault {
	for _, synologyCoupleVault := range synologyCoupleVaults {
		if synologyCoupleVault.Region == region && synologyCoupleVault.Name == vault {
			return synologyCoupleVault
		}
	}
	return nil
}

