package jet

import "context"

// KV key value type
type KV map[string]interface{}

// SortRequest request to sort data
type SortRequest struct {
	Field     string `json:"field"`     // Field which is sorted
	Desc      bool   `json:"desc"`      // Desc if true order by desc
	NullsLast bool   `json:"nullsLast"` // NullsLast if true nulls go last
}

// PagingRequest paging request
type PagingRequest struct {
	Size   int            `json:"size"`   // Size page size
	Index  int            `json:"index"`  // Index page index
	SortBy []*SortRequest `json:"sortBy"` // SortBy sort by fields
}

// PagingRequestG generic paging request
type PagingRequestG[T any] struct {
	PagingRequest
	Request T `json:"request"` // Request
}

// PagingResponse paging response
type PagingResponse struct {
	Total int `json:"total"` // Total number of rows
	Limit int `json:"limit"` // Limit applied
}

// PagingResponseG generic paging response
type PagingResponseG[T any] struct {
	PagingResponse
	Items []*T `json:"response"` // Items
}

// Adapter common interface for adapters
type Adapter[T any] interface {
	// Init initializes adapter
	Init(ctx context.Context, cfg *T) error
	// Close closes storage
	Close(ctx context.Context) error
}

// AdapterListener common interface for adapters with listeners
type AdapterListener[T any] interface {
	Adapter[T]
	// ListenAsync runs async listening
	ListenAsync(ctx context.Context) error
}

// Searchable interface provides a search abilities
type Searchable[TItem any, Rq any] interface {
	// Search searches items by criteria in a pageable manner
	Search(ctx context.Context, rq PagingRequestG[Rq]) (PagingResponseG[TItem], error)
}
