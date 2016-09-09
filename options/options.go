package options

import (
	flag "github.com/spf13/pflag"
	"rsg/loggers"
)

type Options struct {
	AwsId              string
	AwsSecret          string
	Debug              bool
	Dest               string
	Filters            []string
	List               bool
	Region             string
	Vault              string
	RefreshMappingFile *bool
	KeepFiles          *bool
}

func ParseOptions() Options {
	options := Options{}

	flag.StringVarP(&options.Region, "region", "r", "", "region of the vault to restore")
	flag.StringVarP(&options.Vault, "vault", "v", "", "vault to restore")
	flag.BoolVarP(&options.Debug, "debug", "x", false, "display debug info")
	flag.StringSliceVarP(&options.Filters, "filter", "f", []string{}, "filter files to restore (globals * and ?")
	flag.StringVar(&options.AwsId, "aws-id", "", "id of aws credentials")
	flag.StringVar(&options.AwsSecret, "aws-secret", "", "secret of aws credentials")
	flag.StringVarP(&options.Dest, "destination", "d", "", "path to restoration directory")
	flag.BoolVarP(&options.List, "list", "l", false, "list files")
	options.RefreshMappingFile = flag.Bool("refresh-mapping-file", false, "enable or disable refresh of mapping file")
	options.KeepFiles = flag.Bool("keep-files", true, "enable or disable keep existing files")
	flag.Parse()

	if !flag.Lookup("refresh-mapping-file").Changed {
		options.RefreshMappingFile = nil
	}
	if !flag.Lookup("keep-files").Changed {
		options.KeepFiles = nil
	}

	awsIdTruncated := ""
	awsSecretTruncated := ""
	if len(options.AwsId) > 3 {
		awsIdTruncated = options.AwsId[0:3] + "..."
	}
	if len(options.AwsSecret) > 3 {
		awsSecretTruncated = options.AwsSecret[0:3] + "..."
	}

	loggers.DebugFlag = options.Debug
	loggers.Printf(loggers.Debug, "options aws-id: %v\n", awsIdTruncated)
	loggers.Printf(loggers.Debug, "options aws-secret: %v\n", awsSecretTruncated)
	loggers.Printf(loggers.Debug, "options debug: %v\n", options.Debug)
	loggers.Printf(loggers.Debug, "options destination: %v\n", options.Dest)
	loggers.Printf(loggers.Debug, "options filters: %v\n", options.Filters)
	loggers.Printf(loggers.Debug, "options list: %v\n", options.List)
	loggers.Printf(loggers.Debug, "options region: %v\n", options.Region)
	loggers.Printf(loggers.Debug, "options vault: %v\n", options.Vault)
	if options.RefreshMappingFile != nil {
		loggers.Printf(loggers.Debug, "options refresh-mapping-file: %v\n", *options.RefreshMappingFile)
	} else {
		loggers.Print(loggers.Debug, "options refresh-mapping-file: nil\n", )
	}
	if options.KeepFiles != nil {
		loggers.Printf(loggers.Debug, "options keep-files: %v \n", *options.KeepFiles)
	} else {
		loggers.Print(loggers.Debug, "options keep-files: nil\n", )
	}
	return options
}