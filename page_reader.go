package jet

import (
	"context"
)

const (
	defaultPageSize = 100
	chanCapacity    = 10
)

// PageReader is a generic reader allowing read from remote repository
type PageReader[TItem any, TRq any] interface {
	// GetPage retrieves one page
	GetPage(ctx context.Context, rq TRq) chan PagingResponseG[TItem]
}

type pageReader[TItem any, TRq any] struct {
	pageFn   func(context.Context, PagingRequestG[TRq]) (PagingResponseG[TItem], error)
	pageSize int
	logger   CLogger
}

func NewPageReader[TItem any, TRq any](fn func(context.Context, PagingRequestG[TRq]) (PagingResponseG[TItem], error), pageSize int, logger CLogger) PageReader[TItem, TRq] {
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	return &pageReader[TItem, TRq]{
		pageFn:   fn,
		pageSize: pageSize,
		logger:   logger,
	}
}

func (p *pageReader[TItem, TRq]) GetPage(ctx context.Context, rq TRq) chan PagingResponseG[TItem] {
	l := p.logger.C(ctx).Cmp("pager").Mth("get-page")
	res := make(chan PagingResponseG[TItem], chanCapacity)
	go func() {
		pageRequest := PagingRequestG[TRq]{Request: rq}
		pageRequest.Index = 1
		pageRequest.Size = p.pageSize
		defer close(res)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				rs, err := p.pageFn(ctx, pageRequest)
				if err != nil {
					l.E(err).St().ErrF("reading page: %d", pageRequest.Index)
					return
				}
				if len(rs.Items) == 0 {
					return
				}
				pageRequest.Index += 1
				res <- rs
			}
		}
	}()
	return res
}
