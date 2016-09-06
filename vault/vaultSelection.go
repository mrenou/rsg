package vault

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
			loggers.Printf(loggers.Info, "%s:%s\n", synologyCoupleVault.Region, synologyCoupleVault.Name)
		}
		for synologyCoupleVaultToUse == nil {
			region := inputs.QueryString("Select the region of the vault to use for the restoration:")
			vault := inputs.QueryString("Select the vault to use for the restoration:")
			synologyCoupleVaultToUse = getVaultIfExist(region, vault, synologyCoupleVaults)
			if synologyCoupleVaultToUse == nil {
				loggers.Print(loggers.Info, "vault or region doesn't exist. Try again...\n")
			}
		}
	}

	if synologyCoupleVaultToUse != nil {
		loggers.Printf(loggers.Info, "synology backup vault used for the restoration: %s:%s\n", synologyCoupleVaultToUse.Region, synologyCoupleVaultToUse.Name)
		return synologyCoupleVaultToUse.Region, synologyCoupleVaultToUse.Name
	} else {
		loggers.Print(loggers.Error, "no synology backup vault found\n")
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

