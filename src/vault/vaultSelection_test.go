package vault

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"../loggers"
	"../inputs"
	"bytes"
	"os"
	"bufio"
)

func TestSelectRegionVaultFromSynologyVaults_when_there_is_one_synology_vault(t *testing.T) {
	buffer := new(bytes.Buffer)
	loggers.InitLog(os.Stdout, buffer, buffer, buffer)

	synologyVaults := []*SynologyCoupleVault{
		&SynologyCoupleVault{"region1", "vault1", nil, nil},
	}

	region, vault := selectRegionVaultFromSynologyVaults(synologyVaults)

	assert.Equal(t, "region1", region)
	assert.Equal(t, "vault1", vault)
	assert.Equal(t, "synology backup vault used for the restoration: region1:vault1\n", string(buffer.Bytes()))

}

func TestSelectRegionVaultFromSynologyVaults_when_there_is_no_synology_vault(t *testing.T) {
	buffer := new(bytes.Buffer)
	loggers.InitLog(os.Stdout, buffer, buffer, buffer)

	synologyVaults := []*SynologyCoupleVault{

	}

	region, vault := selectRegionVaultFromSynologyVaults(synologyVaults)

	assert.Equal(t, "", region)
	assert.Equal(t, "", vault)
	assert.Equal(t, "ERROR: no synology backup vault found\n", string(buffer.Bytes()))

}

func TestSelectRegionVaultFromSynologyVaults_when_there_is_several_synology_vaults(t *testing.T) {
	buffer := new(bytes.Buffer)
	loggers.InitLog(os.Stdout, buffer, buffer, os.Stdout)
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("region1\nvault1\n")))

	synologyVaults := []*SynologyCoupleVault{
		&SynologyCoupleVault{"region1", "vault1", nil, nil},
		&SynologyCoupleVault{"region1", "vault2", nil, nil},
		&SynologyCoupleVault{"region2", "vault3", nil, nil},
		&SynologyCoupleVault{"region3", "vault1", nil, nil},
	}

	region, vault := selectRegionVaultFromSynologyVaults(synologyVaults)

	assert.Equal(t, "region1", region)
	assert.Equal(t, "vault1", vault)
	assert.Equal(t, "region1:vault1\nregion1:vault2\nregion2:vault3\nregion3:vault1\n" +
	"Select the region of the vault to use for the restoration:" +
	"Select the vault to use for the restoration:" +
	"synology backup vault used for the restoration: region1:vault1\n", string(buffer.Bytes()))

}

func TestSelectRegionVaultFromSynologyVaults_retry_to_give_vault_to_use(t *testing.T) {
	buffer := new(bytes.Buffer)
	loggers.InitLog(os.Stdout, buffer, buffer, os.Stdout)
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("bim\nbam\nregion1\nvault1\n")))

	synologyVaults := []*SynologyCoupleVault{
		&SynologyCoupleVault{"region1", "vault1", nil, nil},
		&SynologyCoupleVault{"region1", "vault2", nil, nil},
		&SynologyCoupleVault{"region2", "vault3", nil, nil},
		&SynologyCoupleVault{"region3", "vault1", nil, nil},
	}

	region, vault := selectRegionVaultFromSynologyVaults(synologyVaults)

	assert.Equal(t, "region1", region)
	assert.Equal(t, "vault1", vault)
	assert.Equal(t, "region1:vault1\nregion1:vault2\nregion2:vault3\nregion3:vault1\n" +
	"Select the region of the vault to use for the restoration:" +
	"Select the vault to use for the restoration:" +
	"vault or region doesn't exist. Try again...\n" +
	"Select the region of the vault to use for the restoration:" +
	"Select the vault to use for the restoration:" +
	"synology backup vault used for the restoration: region1:vault1\n", string(buffer.Bytes()))
}