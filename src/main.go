package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/iam"
	"strings"
	"io/ioutil"
	"./vault"
)

func main() {


	credentialsValue := credentials.NewStaticCredentials("", "", "")
	sessionValue :=  session.New(&aws.Config{Credentials: credentialsValue})

	accountId := getAccountId(sessionValue)

	//glacierClient := glacier.New( sessionValue, &aws.Config{Region: aws.String("eu-west-1")})
	pouet := vault.GetSynologyVaultsOnAllRegions(accountId, sessionValue)
	fmt.Println(pouet)
	//getOuputJob(glacierClient)


}

func getAccountId(sessionValue *session.Session) string {
	svc := iam.New(sessionValue)

	params := &iam.GetUserInput{}
	resp, err := svc.GetUser(params)

	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	return strings.Split(*resp.User.Arn, ":")[4];
}

func listVault(accountId string, glacierClient *glacier.Glacier)   {
	params := &glacier.ListVaultsInput{
		AccountId: aws.String(accountId), // Required
		Limit:     nil,
		Marker:    nil,
	}
	resp, err := glacierClient.ListVaults(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

//{
//VaultList: [{
//CreationDate: "2012-12-23T13:02:43.819Z",
//NumberOfArchives: 0,
//SizeInBytes: 0,
//VaultARN: "arn:aws:glacier:eu-west-1:153914060736:vaults/house",
//VaultName: "house"
//},{
//CreationDate: "2015-03-24T07:35:15.037Z",
//LastInventoryDate: "2016-05-02T09:43:38.769Z",
//NumberOfArchives: 24336,
//SizeInBytes: 130691188845,
//VaultARN: "arn:aws:glacier:eu-west-1:153914060736:vaults/nautilus_0011323B36BD_1",
//VaultName: "nautilus_0011323B36BD_1"
//},{
//CreationDate: "2015-03-24T07:35:15.162Z",
//LastInventoryDate: "2016-04-24T06:56:24.549Z",
//NumberOfArchives: 1,
//SizeInBytes: 10537984,
//VaultARN: "arn:aws:glacier:eu-west-1:153914060736:vaults/nautilus_0011323B36BD_1_mapping",
//VaultName: "nautilus_0011323B36BD_1_mapping"
//}]
//}

func inventoryVault(glacierClient *glacier.Glacier) {
	params := &glacier.InitiateJobInput{
		AccountId: aws.String("153914060736"), // Required
		VaultName: aws.String("nautilus_0011323B36BD_1_mapping"), // Required
		JobParameters: &glacier.JobParameters{
			ArchiveId:   nil,
			Description: aws.String("inventory nautilus_0011323B36BD_1_mapping"),
			Format:      nil,
			InventoryRetrievalParameters: &glacier.InventoryRetrievalJobInput{
				EndDate:   nil,
				Limit:     nil,
				Marker:    nil,
				StartDate: nil,
			},
			RetrievalByteRange: nil,
			SNSTopic:           nil,
			Type:               aws.String("inventory-retrieval"),
		},
	}
	resp, err := glacierClient.InitiateJob(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
}

///usr/local/go/bin/go run /home/morgan/devhome/workspace/perso/test-aws-go/src/main.go
//{
//JobId: "WsYP0WNACtDAhpjtWXUq4pfvmzNs77d1xKKQJJx0YaEe8nmAcWU9FM3CjQ4YU_u2ymL76Zw08TREUwIrczYYO-h6axHq",
//Location: "/153914060736/vaults/nautilus_0011323B36BD_1_mapping/jobs/WsYP0WNACtDAhpjtWXUq4pfvmzNs77d1xKKQJJx0YaEe8nmAcWU9FM3CjQ4YU_u2ymL76Zw08TREUwIrczYYO-h6axHq"
//}


func getOuputJob(glacierClient *glacier.Glacier) {
	params := &glacier.GetJobOutputInput{
		AccountId: aws.String("153914060736"), // Required
		JobId:     aws.String("WsYP0WNACtDAhpjtWXUq4pfvmzNs77d1xKKQJJx0YaEe8nmAcWU9FM3CjQ4YU_u2ymL76Zw08TREUwIrczYYO-h6axHq"), // Required
		VaultName: aws.String("nautilus_0011323B36BD_1_mapping"), // Required
		Range:     nil,
	}
	resp, err := glacierClient.GetJobOutput(params)

	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return
	}

	// Pretty-print the response data.
	fmt.Println(resp)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))

}

