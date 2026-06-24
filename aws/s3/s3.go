package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/zloevil/jet"
	jetaws "github.com/zloevil/jet/aws"
)

const (
	ErrCodeS3PresignPutObject = "S3-001"
	ErrCodeS3DeleteObject     = "S3-002"
	ErrCodeS3GetObject        = "S3-003"
	ErrCodeS3UrlBuildFailed   = "S3-004"
)

var (
	ErrS3PresignPutObject = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeS3PresignPutObject, "presign put object").C(ctx).Wrap(cause).Err()
	}
	ErrS3DeleteObject = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeS3DeleteObject, "delete object").C(ctx).Wrap(cause).Err()
	}
	ErrS3GetObject = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeS3GetObject, "get object").C(ctx).Wrap(cause).Err()
	}
	ErrS3UrlBuildFailed = func(ctx context.Context, cause error) error {
		return jet.NewAppErrBuilder(ErrCodeS3UrlBuildFailed, "url build failed").C(ctx).Wrap(cause).Err()
	}
)

type Config struct {
	PublicBucketName            string `mapstructure:"public_bucket_name"`
	PublicBucketUploadQueueName string `mapstructure:"public_bucket_upload_queue_name"`
	PrivateBucketName           string `mapstructure:"private_bucket_name"`
	PrivateBucketUploadQueue    string `mapstructure:"private_bucket_upload_queue_name"`
	PresignedLinkTTL            int64  `mapstructure:"presigned_link_ttl"`
}

type Client struct {
	logger          jet.CLoggerFunc
	awsCfg          *jetaws.Config
	s3Cfg           *Config
	s3Client        *s3.Client
	s3PresignClient *s3.PresignClient
}

func NewClient(awsCfg *jetaws.Config, s3Cfg *Config, logger jet.CLoggerFunc) *Client {
	return &Client{
		logger: logger,
		awsCfg: awsCfg,
		s3Cfg:  s3Cfg,
	}
}

func (c *Client) l() jet.CLogger {
	return c.logger().Cmp("s3")
}

func (c *Client) Init(ctx context.Context) error {
	awsConfig, err := jetaws.GetAwsConfig(ctx, c.awsCfg)
	if err != nil {
		return err
	}
	c.s3Client = s3.NewFromConfig(*awsConfig)
	c.s3PresignClient = s3.NewPresignClient(c.s3Client)
	return nil
}

func (c *Client) GetNewFileUploadLink(ctx context.Context, private, withPrefix bool, ownerId, fileName, category string) (string, string, error) {
	c.l().C(ctx).Mth("get-new-file-link").Dbg()

	key, err := c.resolveObjectKey(withPrefix, fileName, category, ownerId)
	if err != nil {
		return "", "", err
	}

	req, err := c.s3PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.resolveBucketName(private)),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(c.s3Cfg.PresignedLinkTTL * int64(time.Second))
	})
	if err != nil {
		return "", "", ErrS3PresignPutObject(ctx, err)
	}

	return req.URL, key, nil
}

func (c *Client) GetUpdateFileUploadLink(ctx context.Context, private bool, key string) (string, error) {
	c.l().C(ctx).Mth("get-upd-file-link").Dbg()

	req, err := c.s3PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.resolveBucketName(private)),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(c.s3Cfg.PresignedLinkTTL * int64(time.Second))
	})
	if err != nil {
		return "", ErrS3PresignPutObject(ctx, err)
	}

	return req.URL, nil
}

func (c *Client) GetGetFileLink(ctx context.Context, private bool, key string) (string, error) {
	c.l().C(ctx).Mth("get-file-link").Dbg()

	if !private {
		// public url
		url, err := c.buildPublicUrl(ctx, key)
		if err != nil {
			return "", err
		}
		return url, nil
	}

	// private url
	req, err := c.s3PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.resolveBucketName(private)),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(c.s3Cfg.PresignedLinkTTL * int64(time.Second))
	})
	if err != nil {
		return "", ErrS3PresignPutObject(ctx, err)
	}

	return req.URL, nil
}

func (c *Client) DeleteFileByKey(ctx context.Context, private bool, key string) error {
	c.l().C(ctx).Mth("del-file").Dbg()

	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.resolveBucketName(private)),
		Key:    aws.String(key),
	})
	if err != nil {
		return ErrS3DeleteObject(ctx, err)
	}

	return nil
}

func (c *Client) GetFileByKey(ctx context.Context, private bool, key string) (io.ReadCloser, error) {
	c.l().C(ctx).Mth("del-file").Dbg()

	output, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.resolveBucketName(private)),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, ErrS3GetObject(ctx, err)
	}
	return output.Body, nil
}

func (c *Client) resolveBucketName(isPrivate bool) string {
	if isPrivate {
		return c.s3Cfg.PrivateBucketName
	}
	return c.s3Cfg.PublicBucketName
}

func (c *Client) resolveObjectKey(withPrefix bool, fileName, category, ownerId string) (string, error) {
	if withPrefix {
		return fmt.Sprintf("%s/%s/%s_%s", category, ownerId, jet.NumCode(6), fileName), nil
	}
	return fmt.Sprintf("%s/%s/%s", category, ownerId, fileName), nil
}

func (c *Client) buildPublicUrl(ctx context.Context, key string) (string, error) {
	// Amazon S3 virtual-hosted–style URLs use the following format
	// https://bucket-name.s3.region-code.amazonaws.com/key-name
	// For more details check:
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.resolveBucketName(false), c.awsCfg.Region, pathEscape(key)), nil
}

func pathEscape(path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}
