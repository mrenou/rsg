package core

import (
	"rsg/awsutils"
	"rsg/outputs"
	"github.com/aws/aws-sdk-go/service/glacier"
)

// List aws jobs

func ListJobs(restorationContext *RestorationContext) {
	displayJobsFn := func(page *glacier.ListJobsOutput, lastPage bool) bool {
		for _, desc := range page.JobList {
			outputs.Printfln(outputs.Info, "%s", desc.String())
		}
		return true
	}
	awsutils.DoOnJobPages(restorationContext.GlacierClient, restorationContext.MappingVault, displayJobsFn)
	awsutils.DoOnJobPages(restorationContext.GlacierClient, restorationContext.Vault, displayJobsFn)
}
