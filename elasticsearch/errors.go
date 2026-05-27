package elasticsearch

import (
	"context"
	"github.com/zloevil/jet"
)

var (
	ErrCodeEsNewClient                      = "ES-001"
	ErrCodeEsIdxExists                      = "ES-002"
	ErrCodeEsIdx                            = "ES-003"
	ErrCodeEsIdxAsync                       = "ES-004"
	ErrCodeEsIdxCreate                      = "ES-007"
	ErrCodeEsBulkIdx                        = "ES-008"
	ErrCodeEsExists                         = "ES-009"
	ErrCodeEsInvalidModel                   = "ES-011"
	ErrCodeEsInvalidModelType               = "ES-012"
	ErrCodeEsGetMapping                     = "ES-013"
	ErrCodeEsNoMappingFound                 = "ES-014"
	ErrCodeEsMappingSchemaNotExpected       = "ES-015"
	ErrCodeEsMappingExistentFieldsModified  = "ES-016"
	ErrCodeEsPutMapping                     = "ES-017"
	ErrCodeEsDel                            = "ES-018"
	ErrCodeEsIndexBuilderAliasAndIndexEmpty = "ES-019"
	ErrCodeEsIndexBuilderModelEmpty         = "ES-020"
	ErrCodeEsGetIndexesByAlias              = "ES-021"
	ErrCodeEsNoIndicesForAlias              = "ES-022"
	ErrCodeEsNoWriteIndexForAlias           = "ES-023"
	ErrCodeEsRefresh                        = "ES-024"
	ErrCodeEsBasicAuthInvalid               = "ES-025"
	ErrCodeEsBulkDel                        = "ES-026"
	ErrCodeEsSortRequestFieldEmpty          = "ES-028"
	ErrCodeEsDeleteIdx                      = "ES-029"
)

var (
	ErrEsNewClient = func(cause error) error {
		return jet.NewAppErrBuilder(ErrCodeEsNewClient, "es: new cent").Wrap(cause).Err()
	}
	ErrEsIdxExists = func(ctx context.Context, cause error, index string) error {
		return jet.NewAppErrBuilder(ErrCodeEsIdxExists, "es: index exists (%s)", index).C(ctx).Wrap(cause).Err()
	}
	ErrEsGetMapping = func(ctx context.Context, cause error, index string) error {
		return jet.NewAppErrBuilder(ErrCodeEsGetMapping, "es: get mapping (%s)", index).C(ctx).Wrap(cause).Err()
	}
	ErrEsGetIndexesByAlias = func(ctx context.Context, cause error, alias string) error {
		return jet.NewAppErrBuilder(ErrCodeEsGetIndexesByAlias, "es: get index by alias (%s)", alias).C(ctx).Wrap(cause).Err()
	}
	ErrEsMappingSchemaNotExpected = func(ctx context.Context, cause error, index string) error {
		return jet.NewAppErrBuilder(ErrCodeEsMappingSchemaNotExpected, "es: mapping schema not expected %s", index).C(ctx).F(jet.KV{"idx": index}).Err()
	}
	ErrEsPutMapping = func(ctx context.Context, cause error, index string) error {
		return jet.NewAppErrBuilder(ErrCodeEsPutMapping, "es: mapping").C(ctx).Wrap(cause).F(jet.KV{"idx": index}).Err()
	}
	ErrEsIdx = func(ctx context.Context, cause error, index, id string) error {
		return jet.NewAppErrBuilder(ErrCodeEsIdx, "es: index").C(ctx).Wrap(cause).F(jet.KV{"idx": index, "id": id}).Err()
	}
	ErrEsDel = func(ctx context.Context, cause error, index, id string) error {
		return jet.NewAppErrBuilder(ErrCodeEsDel, "es: delete").C(ctx).Wrap(cause).F(jet.KV{"idx": index, "id": id}).Err()
	}
	ErrEsRefresh = func(ctx context.Context, cause error, index string) error {
		return jet.NewAppErrBuilder(ErrCodeEsRefresh, "es: refresh").C(ctx).Wrap(cause).F(jet.KV{"idx": index}).Err()
	}
	ErrEsBulkIdx = func(ctx context.Context, cause error, index string) error {
		return jet.NewAppErrBuilder(ErrCodeEsBulkIdx, "es: bulk index").C(ctx).Wrap(cause).F(jet.KV{"idx": index}).Err()
	}
	ErrEsDeleteIdx = func(ctx context.Context, cause error, index string) error {
		return jet.NewAppErrBuilder(ErrCodeEsDeleteIdx, "es: delete index").C(ctx).Wrap(cause).F(jet.KV{"idx": index}).Err()
	}
	ErrEsBulkDel = func(ctx context.Context, cause error, index string) error {
		return jet.NewAppErrBuilder(ErrCodeEsBulkDel, "e: bulk delete").C(ctx).Wrap(cause).F(jet.KV{"idx": index}).Err()
	}
	ErrEsIdxCreate = func(ctx context.Context, cause error, index string) error {
		return jet.NewAppErrBuilder(ErrCodeEsIdxCreate, "es: create").C(ctx).Wrap(cause).F(jet.KV{"idx": index}).Err()
	}
	ErrEsExists = func(ctx context.Context, cause error, index, id string) error {
		return jet.NewAppErrBuilder(ErrCodeEsExists, "es: exsits").C(ctx).Wrap(cause).F(jet.KV{"idx": index, "id": id}).Err()
	}
	ErrEsNoMappingFound = func(ctx context.Context, index string) error {
		return jet.NewAppErrBuilder(ErrCodeEsNoMappingFound, "no mapping found").C(ctx).F(jet.KV{"idx": index}).Err()
	}
	ErrEsMappingExistentFieldsModified = func(ctx context.Context, index string, fields []string) error {
		return jet.NewAppErrBuilder(ErrCodeEsMappingExistentFieldsModified, "ES doesn't allow changing mapping for existent fields.").C(ctx).F(jet.KV{"idx": index, "fields": fields}).Err()
	}
	ErrEsInvalidModel = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeEsInvalidModel, "invalid model, check tags").C(ctx).Err()
	}
	ErrEsInvalidModelType = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeEsInvalidModelType, "model must be pointer of struct").C(ctx).Err()
	}
	ErrEsIndexBuilderAliasAndIndexEmpty = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeEsIndexBuilderAliasAndIndexEmpty, "neither alias name nor index name specified").C(ctx).Err()
	}
	ErrEsIndexBuilderModelEmpty = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeEsIndexBuilderModelEmpty, "model not specified").C(ctx).Err()
	}
	ErrEsNoIndicesForAlias = func(ctx context.Context, alias string) error {
		return jet.NewAppErrBuilder(ErrCodeEsNoIndicesForAlias, "model not specified").F(jet.KV{"alias": alias}).C(ctx).Err()
	}
	ErrEsNoWriteIndexForAlias = func(ctx context.Context, alias string) error {
		return jet.NewAppErrBuilder(ErrCodeEsNoWriteIndexForAlias, "no write index").F(jet.KV{"alias": alias}).C(ctx).Err()
	}
	ErrEsBasicAuthInvalid = func() error {
		return jet.NewAppErrBuilder(ErrCodeEsBasicAuthInvalid, "basic auth invalid").Err()
	}
	ErrEsSortRequestFieldEmpty = func(ctx context.Context) error {
		return jet.NewAppErrBuilder(ErrCodeEsSortRequestFieldEmpty, "sort request field parameter empty").C(ctx).Err()
	}
)
