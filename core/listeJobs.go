package core

import (
	"rsg/awsutils"
	"rsg/loggers"
	"github.com/aws/aws-sdk-go/service/glacier"
)

func ListJobs(restorationContext *awsutils.RestorationContext) {
	displayJobsFn := func(page *glacier.ListJobsOutput, lastPage bool) bool {
		for _, desc := range page.JobList {
			loggers.Printfln(loggers.Info, "%s", desc.String())
		}
		return true
	}
	awsutils.DoOnJobPages(restorationContext, restorationContext.MappingVault, displayJobsFn)
	awsutils.DoOnJobPages(restorationContext, restorationContext.Vault, displayJobsFn)
}
