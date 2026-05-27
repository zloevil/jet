package elasticsearch

import (
	"context"
	"github.com/olivere/elastic/v7"
	"github.com/zloevil/jet"
)

func ToSortRequestEs(ctx context.Context, request *jet.SortRequest) (*elastic.SortInfo, error) {

	if request.Field == "" {
		return nil, ErrEsSortRequestFieldEmpty(ctx)
	}

	res := EsSortRequestMissingFirst
	if request.NullsLast {
		res = EsSortRequestMissingLast
	}

	return &elastic.SortInfo{
		Field:     request.Field,
		Ascending: !request.Desc,
		Missing:   res,
	}, nil
}
