//go:build dev

package s3

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	jetaws "github.com/zloevil/jet/aws"
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
	s3Cfg = &Config{
		PublicBucketName:  "ext.storage.dev.back",
		PrivateBucketName: "int.storage.dev.back",
		PresignedLinkTTL:  60,
	}
	awsCfg = &jetaws.Config{
		Region:              "eu-central-1",
		AccessKeyId:         "access_key_id",
		SecretAccessKey:     "secret_access_key",
		SharedConfigProfile: "back/dev",
	}
)

func (s *s3TestSuite) Test_S3() {

	// init client
	client := NewClient(awsCfg, s3Cfg, s.logger)
	s.NoError(client.Init(s.Ctx))
	s.NotEmpty(client.s3Client)

	// get new upload link
	ownerId := jet.NewId()
	fn := fmt.Sprintf("%s.png", jet.NewRandString())
	url, key, err := client.GetNewFileUploadLink(s.Ctx, false, false, ownerId, fn, "test")
	s.NoError(err)
	s.NotEmpty(url)
	s.NotEmpty(key)

	// update
	url, err = client.GetUpdateFileUploadLink(s.Ctx, false, key)
	s.NoError(err)
	s.NotEmpty(url)

	// get
	url, err = client.GetGetFileLink(s.Ctx, false, key)
	s.NoError(err)
	s.NotEmpty(url)
	s.L().DbgF("url: %s", url)

	// delete
	s.NoError(client.DeleteFileByKey(s.Ctx, false, key))
}

func (s *s3TestSuite) Test_S3_PublicUrlEscape() {
	// init client
	client := NewClient(awsCfg, s3Cfg, s.logger)
	s.NoError(client.Init(s.Ctx))
	s.NotEmpty(client.s3Client)

	ownerId := jet.NewId()
	fn := getFilenameThatRequiresUrlEscape()

	// get new upload link
	url, key, err := client.GetNewFileUploadLink(s.Ctx, false, false, ownerId, fn, "test")
	s.L().DbgF("public url: %s", url)
	s.NoError(err)
	s.NotEmpty(key)
	s.NotEmpty(url)
	s.True(isURLEscaped(url))

	url, key, err = client.GetNewFileUploadLink(s.Ctx, true, false, ownerId, fn, "test")
	s.L().DbgF("private url: %s", url)
	s.NoError(err)
	s.NotEmpty(key)
	s.NotEmpty(url)
	s.True(isURLEscaped(url))

	// update
	url, err = client.GetUpdateFileUploadLink(s.Ctx, false, key)
	s.L().DbgF("public url: %s", url)
	s.NoError(err)
	s.NotEmpty(url)
	s.True(isURLEscaped(url))

	url, err = client.GetUpdateFileUploadLink(s.Ctx, true, key)
	s.L().DbgF("private url: %s", url)
	s.NoError(err)
	s.NotEmpty(url)
	s.True(isURLEscaped(url))

	// get
	url, err = client.GetGetFileLink(s.Ctx, false, key)
	s.L().DbgF("public url: %s", url)
	s.NoError(err)
	s.NotEmpty(url)
	s.True(isURLEscaped(url))

	url, err = client.GetGetFileLink(s.Ctx, true, key)
	s.L().DbgF("private url: %s", url)
	s.NoError(err)
	s.NotEmpty(url)
	s.True(isURLEscaped(url))
}

// getFilenameThatRequiresUrlEscape gets filename that requires URL-escape in case being placed in URL directly
func getFilenameThatRequiresUrlEscape() string {
	return fmt.Sprintf("{#%s#}.png", jet.NewRandString())
}

// isURLEscaped returns true if the input looks like it's already URL-escaped
func isURLEscaped(s string) bool {
	decoded, err := url.QueryUnescape(s)
	if err != nil {
		return false
	}
	// if decoding changes the string, the string was already escaped
	return decoded != s
}
