package minio

import (
	"context"
	"github.com/zloevil/jet"
)

const (
	ErrCodeErrMinioPutObject        = "S3-001"
	ErrCodeMinioCannotGetObject     = "S3-002"
	ErrCodeMinioCannotGetStatObject = "S3-003"
	ErrCodeMinioObjectNotFound      = "S3-004"
	ErrCodeMinioCreateBucket        = "S3-005"
	ErrCodeMinioRemoveObject        = "S3-006"
	ErrCodeMinioNew                 = "S3-007"
	ErrCodeMinioCopyObject          = "S3-008"
)

var (
	ErrMinioPutObject = func(cause error, ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeErrMinioPutObject, "").Wrap(cause).C(ctx).Err()
	}
	ErrMinioCannotGetObject = func(cause error, ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeMinioCannotGetObject, "").Wrap(cause).C(ctx).Err()
	}
	ErrMinioCannotGetStatObject = func(cause error, ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeMinioCannotGetStatObject, "").Wrap(cause).C(ctx).Err()
	}
	ErrMinioObjectNotFound = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeMinioObjectNotFound, "").C(ctx).Err()
	}
	ErrMinioCreateBucket = func(cause error, ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeMinioCreateBucket, "").Wrap(cause).C(ctx).Err()
	}
	ErrMinioRemoveObject = func(cause error, ctx context.Context, fileId string) error {
		return jet.NewAppErrBuilder(ErrCodeMinioRemoveObject, "").Wrap(cause).C(ctx).F(jet.KV{"fileID ": fileId}).Err()
	}
	ErrMinioCopyObject = func(cause error, ctx context.Context, fileId string) error {
		return jet.NewAppErrBuilder(ErrCodeMinioCopyObject, "").Wrap(cause).C(ctx).F(jet.KV{"fileID ": fileId}).Err()
	}
	ErrMinioNew = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeMinioNew, "").Wrap(cause).Err()
	}
)
