package aerospike

import (
	"context"
	"github.com/aerospike/aerospike-client-go/v8"
)

func AsString(ctx context.Context, bm aerospike.BinMap, bin string) (string, error) {
	if bm == nil {
		return "", nil
	}
	v, ok := bm[bin]
	if !ok || v == nil {
		return "", nil
	}
	r, ok := v.(string)
	if !ok {
		return "", ErrAeroInvalidBinType(ctx, bin)
	}
	return r, nil
}

func AsFloat(ctx context.Context, bm aerospike.BinMap, bin string) (float64, error) {
	if bm == nil {
		return 0.0, nil
	}
	v, ok := bm[bin]
	if !ok || v == nil {
		return 0.0, nil
	}
	r, ok := v.(float64)
	if !ok {
		return 0.0, ErrAeroInvalidBinType(ctx, bin)
	}
	return r, nil
}

func AsInt(ctx context.Context, bm aerospike.BinMap, bin string) (int, error) {
	if bm == nil {
		return 0, nil
	}
	v, ok := bm[bin]
	if !ok || v == nil {
		return 0, nil
	}
	r, ok := v.(int)
	if !ok {
		return 0.0, ErrAeroInvalidBinType(ctx, bin)
	}
	return r, nil
}

func AsBool(ctx context.Context, bm aerospike.BinMap, bin string) (bool, error) {
	if bm == nil {
		return false, nil
	}
	v, ok := bm[bin]
	if !ok || v == nil {
		return false, nil
	}
	r, ok := v.(bool)
	if !ok {
		return false, ErrAeroInvalidBinType(ctx, bin)
	}
	return r, nil
}

func AsStrings(ctx context.Context, bm aerospike.BinMap, bin string) ([]string, error) {
	if bm == nil {
		return nil, nil
	}
	v, ok := bm[bin]
	if !ok || v == nil {
		return nil, nil
	}
	strs, ok := v.([]interface{})
	if !ok {
		return nil, ErrAeroInvalidBinType(ctx, bin)
	}
	var r []string
	for _, i := range strs {
		vi, ok := i.(string)
		if ok {
			r = append(r, vi)
		}
	}
	return r, nil
}

func AsBytes(ctx context.Context, bm aerospike.BinMap, bin string) ([]byte, error) {
	if bm == nil {
		return nil, nil
	}
	v, ok := bm[bin]
	if !ok || v == nil {
		return nil, nil
	}
	r, ok := v.([]byte)
	if !ok {
		return nil, ErrAeroInvalidBinType(ctx, bin)
	}
	return r, nil
}
