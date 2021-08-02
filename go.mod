module github.com/mpostument/ebs-autoresize

go 1.15

require (
	github.com/aws/aws-sdk-go-v2 v1.7.1
	github.com/aws/aws-sdk-go-v2/config v1.1.5
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.0.6
	github.com/aws/aws-sdk-go-v2/internal/ini v1.1.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.4.0
	github.com/aws/smithy-go v1.6.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mvisonneau/go-ebsnvme v0.0.0-20201026165225-e63797fabc2f
	github.com/shirou/gopsutil/v3 v3.21.3
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	golang.org/x/text v0.3.3 // indirect
)
