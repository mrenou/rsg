package options

import (
	flag "github.com/spf13/pflag"
	"rsg/loggers"
)

type Options struct {
	AwsId              string
	AwsSecret          string
	Verbose            bool
	Dest               string
	Filters            []string
	List               bool
	Region             string
	Vault              string
	InfoMessage        bool
	RefreshMappingFile *bool
	KeepFiles          *bool
}

func ParseOptions() Options {
	options := Options{}

	flag.StringVarP(&options.Region, "region", "r", "", "region of the vault to restore")
	flag.StringVarP(&options.Vault, "vault", "v", "", "vault to restore")
	flag.BoolVar(&options.Verbose, "verbose", false, "display low level messages")
	flag.StringSliceVarP(&options.Filters, "filter", "f", []string{}, "filter files to restore (globals * and ?)")
	flag.StringVar(&options.AwsId, "aws-id", "", "id of aws credentials")
	flag.StringVar(&options.AwsSecret, "aws-secret", "", "secret of aws credentials")
	flag.StringVarP(&options.Dest, "destination", "d", "", "path to restoration directory")
	flag.BoolVarP(&options.List, "list", "l", false, "list files")
	flag.BoolVar(&options.InfoMessage, "info-messages", true, "display information messages")
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

	loggers.VerboseFlag = options.Verbose
	loggers.OptionalInfoFlag = options.InfoMessage
	loggers.Printf(loggers.Verbose, "Options aws-id: %v\n", awsIdTruncated)
	loggers.Printf(loggers.Verbose, "Options aws-secret: %v\n", awsSecretTruncated)
	loggers.Printf(loggers.Verbose, "Options destination: %v\n", options.Dest)
	loggers.Printf(loggers.Verbose, "Options filters: %v\n", options.Filters)
	if options.KeepFiles != nil {
		loggers.Printf(loggers.Verbose, "Options keep-files: %v \n", *options.KeepFiles)
	} else {
		loggers.Print(loggers.Verbose, "Options keep-files: nil\n", )
	}
	loggers.Printf(loggers.Verbose, "Options list: %v\n", options.List)
	loggers.Printf(loggers.Verbose, "Options info-messages: %v\n", options.InfoMessage)
	if options.RefreshMappingFile != nil {
		loggers.Printf(loggers.Verbose, "Options refresh-mapping-file: %v\n", *options.RefreshMappingFile)
	} else {
		loggers.Print(loggers.Verbose, "Options refresh-mapping-file: nil\n", )
	}
	loggers.Printf(loggers.Verbose, "Options region: %v\n", options.Region)
	loggers.Printf(loggers.Verbose, "Options vault: %v\n", options.Vault)
	loggers.Printf(loggers.Verbose, "Options verbose: %v\n", options.Verbose)
	return options
}