package core

import (
	"github.com/aws/aws-sdk-go/aws"
	"rsg/outputs"
	"rsg/utils"
	"github.com/aws/aws-sdk-go/service/glacier"
	"os/user"
	"io/ioutil"
	"encoding/json"
	"os"
	"github.com/aws/aws-sdk-go/service/glacier/glacieriface"
	"rsg/options"
	"rsg/awsutils"
)

type RestorationContext struct {
	GlacierClient        glacieriface.GlacierAPI
	WorkingDirPath       string
	Region               string
	Vault                string
	MappingVault         string
	RegionVaultCache     RegionVaultCache
	DestinationDirPath   string
	BytesBySecond        uint64
	Options              RestorationOptions
}

type RestorationOptions struct {
	Filters            []string
	RefreshMappingFile *bool
	KeepFiles          *bool
	InfoMessage        bool
}

type RegionVaultCache struct {
	MappingArchive             *awsutils.Archive
}

func CreateRestorationContext(region, vault string, optionsValue options.Options) *RestorationContext {
	usr, err := user.Current()
	utils.ExitIfError(err)
	workingDirPath := usr.HomeDir + "/.rsg/" + region + "/" + vault
	err = os.MkdirAll(workingDirPath, 0700)
	utils.ExitIfError(err)
	glacierClient := glacier.New(awsutils.Session, &aws.Config{Region: aws.String(region)})
	cache := ReadRegionVaultCache(region, vault, workingDirPath);
	return &RestorationContext{GlacierClient: glacierClient,
		WorkingDirPath: workingDirPath,
		Region: region,
		Vault: vault,
		MappingVault: vault + "_mapping",
		RegionVaultCache: cache,
		DestinationDirPath: optionsValue.Dest,
		BytesBySecond: 0,
		Options: RestorationOptions{Filters: optionsValue.Filters,
			RefreshMappingFile: optionsValue.RefreshMappingFile,
			KeepFiles: optionsValue.KeepFiles,
			InfoMessage: optionsValue.InfoMessage,
		},
	}
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
	outputs.Println(outputs.Verbose, "Write cache")
	bytes, err := json.Marshal(restorationContext.RegionVaultCache)
	utils.ExitIfError(err)
	err = ioutil.WriteFile(restorationContext.WorkingDirPath + "/cache.json", bytes, 0700)
	utils.ExitIfError(err)
}

func (restorationContext *RestorationContext) GetMappingFilePath() string {
	return restorationContext.WorkingDirPath + "/mapping.sqllite"
}