package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/zloevil/jet"
)

const (
	ErrCodeAwsLoadDefConfig = "AWS-001"
)

var (
	ErrAwsLoadDefConfig = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeAwsLoadDefConfig, "load aws config").C(ctx).Wrap(cause).Err()
	}
)

type Config struct {
	Region              string `mapstructure:"region"`
	AccessKeyId         string `mapstructure:"access_key_id"`
	SecretAccessKey     string `mapstructure:"secret_access_key"`
	SharedConfigProfile string `mapstructure:"shared_config_profile"`
}

func GetAwsConfig(ctx context.Context, configuration *Config) (*aws.Config, error) {
	var cfg aws.Config
	var err error
	if configuration.SharedConfigProfile == "" {
		cfg = aws.Config{
			Region: configuration.Region,
			Credentials: credentials.StaticCredentialsProvider{Value: aws.Credentials{
				AccessKeyID:     configuration.AccessKeyId,
				SecretAccessKey: configuration.SecretAccessKey,
			}},
		}
	} else {
		cfg, err = awsCfg.LoadDefaultConfig(ctx, awsCfg.WithSharedConfigProfile(configuration.SharedConfigProfile))
		if err != nil {
			return nil, ErrAwsLoadDefConfig(ctx, err)
		}
	}
	return &cfg, nil
}
