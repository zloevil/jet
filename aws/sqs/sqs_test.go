//go:build dev

package sqs

import (
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	jetaws "github.com/zloevil/jet/aws"
	"testing"
)

type s3TestSuite struct {
	jet.Suite
	logger jet.CLoggerFunc
}

func (s *s3TestSuite) SetupSuite() {
	s.logger = func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) }
	s.Suite.Init(s.logger)
}

func TestS3Suite(t *testing.T) {
	suite.Run(t, new(s3TestSuite))
}

var (
	awsCfg = &jetaws.Config{
		Region:              "eu-central-1",
		AccessKeyId:         "access_key_id",
		SecretAccessKey:     "secret_access_key",
		SharedConfigProfile: "back/dev",
	}
)

func (s *s3TestSuite) Test_Init() {
	// init client
	client := NewClient(awsCfg, s.logger)
	s.NoError(client.Init(s.Ctx))
	s.NotEmpty(client.sqsClient)

	_, err := client.GetQueueURL(s.Ctx, &sqs.GetQueueUrlInput{
		QueueName:              jet.StringPtr("ext-storage-dev"),
		QueueOwnerAWSAccountId: nil,
	})
	s.NoError(err)
}
