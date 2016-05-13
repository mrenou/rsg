package awsutils

import (
	"github.com/aws/aws-sdk-go/aws"
	"../loggers"
	"../utils"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/aws/session"
	"os/user"
	"io/ioutil"
	"encoding/json"
	"os"
	"github.com/aws/aws-sdk-go/service/glacier/glacieriface"
)

type Archive struct {
	ArchiveId string
	Size      uint64
}

type RestorationContext struct {
	GlacierClient    glacieriface.GlacierAPI
	WorkingDirPath    string
	Region           string
	Vault            string
	MappingVault     string
	AccountId        string
	RegionVaultCache RegionVaultCache;
}

type RegionVaultCache struct {
	MappingVaultInventoryJobId string
	MappingArchive             *Archive
	MappingVaultRetrieveJobId  string
}

func CreateRestorationContext(sessionValue *session.Session, accountId, region, vault string) *RestorationContext {
	usr, err := user.Current()
	utils.ExitIfError(err)
	workingDirPath :=  usr.HomeDir + "/.rsg/" + region + "/" + vault
	err = os.MkdirAll(workingDirPath, 0700)
	utils.ExitIfError(err)
	glacierClient := glacier.New(sessionValue, &aws.Config{Region: aws.String(region)})
	cache := ReadRegionVaultCache(region, vault, workingDirPath);
	return &RestorationContext{glacierClient, workingDirPath, region, vault, vault + "_mapping", accountId, cache}
}

func ReadRegionVaultCache(region, vault, workingDirPath string) RegionVaultCache {
	if bytes, err := ioutil.ReadFile(workingDirPath + "/cache.json"); err == nil {
		cache := RegionVaultCache{}
		err = json.Unmarshal(bytes, &cache)
		utils.ExitIfError(err)
		// TODO warning ?
		return cache
	} else {
		return RegionVaultCache{}
	}
}

func (restorationContext *RestorationContext) WriteRegionVaultCache() {
	loggers.DebugPrintf("write cache %v\n", restorationContext.RegionVaultCache)
	bytes, err := json.Marshal(restorationContext.RegionVaultCache)
	utils.ExitIfError(err)
	err = ioutil.WriteFile(restorationContext.WorkingDirPath + "/cache.json", bytes, 0700)
	utils.ExitIfError(err)
}

func (restorationContext *RestorationContext) GetMappingFilePath() string {
	return restorationContext.WorkingDirPath + "/mapping.sqllite"
}