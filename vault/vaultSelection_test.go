package vault

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"rsg/loggers"
	"rsg/inputs"
	"bytes"
	"os"
	"bufio"
	"rsg/consts"
)

func TestSelectRegionVaultFromSynologyVaults_when_there_is_one_synology_vault(t *testing.T) {
	buffer := new(bytes.Buffer)
	loggers.InitLog(os.Stdout, buffer, buffer, buffer, buffer)

	synologyVaults := []*SynologyCoupleVault{
		&SynologyCoupleVault{"region1", "vault1", nil, nil},
	}

	region, vault := selectRegionVaultFromSynologyVaults(synologyVaults)

	assert.Equal(t, "region1", region)
	assert.Equal(t, "vault1", vault)
	assert.Equal(t, "Synology backup vault used: region1:vault1" + consts.LINE_BREAK, string(buffer.Bytes()))

}

func TestSelectRegionVaultFromSynologyVaults_when_there_is_no_synology_vault(t *testing.T) {
	buffer := new(bytes.Buffer)
	loggers.InitLog(os.Stdout, buffer, buffer, buffer, buffer)

	synologyVaults := []*SynologyCoupleVault{

	}

	region, vault := selectRegionVaultFromSynologyVaults(synologyVaults)

	assert.Equal(t, "", region)
	assert.Equal(t, "", vault)
	assert.Equal(t, "ERROR: No synology backup vault found" + consts.LINE_BREAK, string(buffer.Bytes()))

}

func TestSelectRegionVaultFromSynologyVaults_when_there_is_several_synology_vaults(t *testing.T) {
	buffer := new(bytes.Buffer)
	loggers.InitLog(os.Stdout, buffer, buffer, buffer, os.Stdout)
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("region1" + consts.LINE_BREAK + "vault1" + consts.LINE_BREAK)))

	synologyVaults := []*SynologyCoupleVault{
		&SynologyCoupleVault{"region1", "vault1", nil, nil},
		&SynologyCoupleVault{"region1", "vault2", nil, nil},
		&SynologyCoupleVault{"region2", "vault3", nil, nil},
		&SynologyCoupleVault{"region3", "vault1", nil, nil},
	}

	region, vault := selectRegionVaultFromSynologyVaults(synologyVaults)

	assert.Equal(t, "region1", region)
	assert.Equal(t, "vault1", vault)
	assert.Equal(t, "region1:vault1" + consts.LINE_BREAK + "region1:vault2" + consts.LINE_BREAK + "region2:vault3" + consts.LINE_BREAK + "region3:vault1" + consts.LINE_BREAK +
	"Select the region of the vault to use: " +
	"Select the vault to use: " +
	"Synology backup vault used: region1:vault1" + consts.LINE_BREAK, string(buffer.Bytes()))

}

func TestSelectRegionVaultFromSynologyVaults_retry_to_give_vault_to_use(t *testing.T) {
	buffer := new(bytes.Buffer)
	loggers.InitLog(os.Stdout, buffer, buffer, buffer, os.Stdout)
	inputs.StdinReader = bufio.NewReader(bytes.NewReader([]byte("bim" + consts.LINE_BREAK + "bam" + consts.LINE_BREAK + "region1" + consts.LINE_BREAK + "vault1" + consts.LINE_BREAK)))

	synologyVaults := []*SynologyCoupleVault{
		&SynologyCoupleVault{"region1", "vault1", nil, nil},
		&SynologyCoupleVault{"region1", "vault2", nil, nil},
		&SynologyCoupleVault{"region2", "vault3", nil, nil},
		&SynologyCoupleVault{"region3", "vault1", nil, nil},
	}

	region, vault := selectRegionVaultFromSynologyVaults(synologyVaults)

	assert.Equal(t, "region1", region)
	assert.Equal(t, "vault1", vault)
	assert.Equal(t, "region1:vault1" + consts.LINE_BREAK + "region1:vault2" + consts.LINE_BREAK + "region2:vault3" + consts.LINE_BREAK + "region3:vault1" + consts.LINE_BREAK +
	"Select the region of the vault to use: " +
	"Select the vault to use: " +
	"Vault or region doesn't exist. Try again..." + consts.LINE_BREAK +
	"Select the region of the vault to use: " +
	"Select the vault to use: " +
	"Synology backup vault used: region1:vault1" + consts.LINE_BREAK, string(buffer.Bytes()))
}