package minio

import (
	"context"
	"fmt"
	"github.com/go-viper/mapstructure/v2"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/zloevil/jet"
	"io"
)

type Minio struct {
	logger   jet.CLoggerFunc
	Instance *minio.Client
}

// FileInfo is a meta info about stored file
type FileInfo struct {
	Id           string            `json:"id"`          // Id file ID
	Filename     string            `json:"fileName"`    // Filename - file name
	BucketName   string            `json:"bucket"`      // BucketName - bucket name (like storage folder)
	Extension    string            `json:"ext"`         // Extension - file ext
	LastModified string            `json:"modified"`    // LastModified - last modified date
	Size         int64             `json:"size"`        // Size - length of a file in bytes
	ContentType  string            `json:"contentType"` // ContentType - content type
	Metadata     map[string]string `json:"metadata"`    // Metadata - some additional user params attached to a stored file
}

// Config minio config
type Config struct {
	Host      string
	Port      string
	AccessKey string `config:"access-key"`
	SecretKey string `config:"secret-key"`
	Ssl       bool
}

func (a *Minio) l() jet.CLogger {
	return a.logger().Cmp("minio")
}

func New(params *Config, logger jet.CLoggerFunc) (*Minio, error) {
	client, err := minio.New(fmt.Sprintf("%s:%s", params.Host, params.Port), &minio.Options{
		Creds:  credentials.NewStaticV4(params.AccessKey, params.SecretKey, ""),
		Secure: params.Ssl,
	})
	if err != nil {
		return nil, ErrMinioNew(err)
	}

	return &Minio{
		Instance: client,
		logger:   logger,
	}, nil
}

func (a *Minio) fileInfoToMap(info *FileInfo) map[string]string {
	result := map[string]string{}
	mp := map[string]interface{}{}
	_ = mapstructure.WeakDecode(info, &mp)
	delete(mp, "Metadata")
	for k, v := range info.Metadata {
		result[k] = v
	}
	for k, v := range mp {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

func (a *Minio) mapToFileInfo(meta map[string]string) *FileInfo {
	result := &FileInfo{
		Metadata: map[string]string{},
	}
	ms := &mapstructure.Metadata{}
	_ = mapstructure.WeakDecodeMetadata(meta, &result, ms)
	for i := range ms.Unused {
		result.Metadata[ms.Unused[i]] = meta[ms.Unused[i]]
	}
	return result
}

func (a *Minio) Put(ctx context.Context, fi *FileInfo, file io.Reader) error {
	l := a.l().Mth("put-file").C(ctx).Dbg()

	// size -1 enables default behaviour
	// if we have real size it's better to pass it
	var size int64 = -1
	if fi.Size > 0 {
		size = fi.Size
	}
	info, err := a.Instance.PutObject(ctx, fi.BucketName, fi.Id, file, size, minio.PutObjectOptions{
		UserMetadata: a.fileInfoToMap(fi),
	})
	if err != nil {
		return ErrMinioPutObject(err, ctx)
	}
	l.F(jet.KV{"key": info.Key})
	return nil
}

func (a *Minio) Get(ctx context.Context, bucketName string, fileID string) (io.Reader, error) {
	a.l().Mth("get-file").C(ctx).Dbg()
	object, err := a.Instance.GetObject(ctx, bucketName, fileID, minio.GetObjectOptions{})
	if err != nil {
		return nil, ErrMinioCannotGetObject(err, ctx)
	}
	stat, err := object.Stat()
	if err != nil {
		return nil, ErrMinioCannotGetStatObject(err, ctx)
	}
	if stat.Key != fileID {
		return nil, ErrMinioObjectNotFound(ctx)
	}
	return object, nil
}

func (a *Minio) GetMetadata(ctx context.Context, bucketName string, fileID string) (*FileInfo, error) {
	a.l().Mth("get-metadata").C(ctx).Dbg()
	stat, err := a.Instance.StatObject(ctx, bucketName, fileID, minio.StatObjectOptions{})
	if err != nil {
		return nil, ErrMinioCannotGetStatObject(err, ctx)
	}
	if stat.Key != fileID {
		return nil, ErrMinioObjectNotFound(ctx)
	}
	return a.mapToFileInfo(stat.UserMetadata), nil
}

func (a *Minio) IsBucketExist(ctx context.Context, bucketName string) bool {
	a.l().Mth("is-bucket-exist").C(ctx).Dbg()
	exist, _ := a.Instance.BucketExists(ctx, bucketName)
	return exist
}

func (a *Minio) CreateBucket(ctx context.Context, bucketName string) error {
	a.l().Mth("create-bucket").C(ctx).Dbg()
	err := a.Instance.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		return ErrMinioCreateBucket(err, ctx)
	}
	return nil
}

func (a *Minio) Delete(ctx context.Context, bucketName string, fileID string) error {
	a.l().Mth("delete").C(ctx).Dbg()
	err := a.Instance.RemoveObject(ctx, bucketName, fileID, minio.RemoveObjectOptions{})
	if err != nil {
		return ErrMinioRemoveObject(err, ctx, fileID)
	}
	return nil
}

func (a *Minio) IsFileExist(ctx context.Context, bucketName string, fileID string) bool {
	a.l().Mth("is-file-exists").C(ctx).Dbg()
	stat, err := a.Instance.StatObject(ctx, bucketName, fileID, minio.StatObjectOptions{})
	return err == nil && stat.Key == fileID
}

func (a *Minio) Copy(ctx context.Context, srcBucketName, srcFileID, destBucketName, destFileId string) error {
	a.l().Mth("copy").C(ctx).Dbg()
	_, err := a.Instance.CopyObject(ctx, minio.CopyDestOptions{Bucket: destBucketName, Object: destFileId}, minio.CopySrcOptions{Bucket: srcBucketName, Object: srcFileID})
	if err != nil {
		return ErrMinioCopyObject(err, ctx, srcFileID)
	}
	return nil
}
