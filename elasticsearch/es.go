package elasticsearch

import (
	"context"
	"github.com/olivere/elastic/v7"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/goroutine"
)

// Config - model of ES configuration
type Config struct {
	Url      string // Url - ES url
	Trace    bool   // Trace enables tracing mode
	Sniff    bool   // Sniff - read https://github.com/olivere/elastic/issues/387
	Shards   int    // Shards - how many shards to be created for index
	Replicas int    // Replicas - how many replicas to eb created for index
	Username string // Username - ES basic auth (if not set, no auth applied)
	Password string // Password - ES basic auth
	Refresh  bool   // Refresh - enforces refresh after each change. It helpful for tests but MUST NOT BE USED ON PROD
}

// Search allows indexing and searching with ES
type Search interface {
	// Index indexes a document
	Index(ctx context.Context, index string, id string, data interface{}) error
	// IndexAsync indexes a document async
	IndexAsync(ctx context.Context, index string, id string, data interface{})
	// IndexBulk allows indexing bulk of documents in one hit
	IndexBulk(ctx context.Context, index string, docs map[string]interface{}) error
	// IndexBulkAsync allows indexing bulk of documents in one hit
	IndexBulkAsync(ctx context.Context, index string, docs map[string]interface{})
	// GetClient provides an access to ES client
	GetClient() *elastic.Client
	// Close closes client
	Close(ctx context.Context)
	//Ping pings server
	Ping() bool
	// Exists checks if a document exists in the index
	Exists(ctx context.Context, index, id string) (bool, error)
	// Delete deletes a document
	Delete(ctx context.Context, index string, id string) error
	// DeleteBulk deletes bulk of documents
	DeleteBulk(ctx context.Context, index string, ids []string) error
	// NewBuilder creates a new builder object
	NewBuilder() IndexBuilder
	// Refresh refreshes data in index (don't use in production code)
	Refresh(ctx context.Context, index string) error
	// DeleteIndex removes index
	DeleteIndex(ctx context.Context, index string) error
}

type esImpl struct {
	client *elastic.Client
	logger jet.CLoggerFunc
	cfg    *Config
	url    string
}

func (s *esImpl) l() jet.CLogger {
	return s.logger().Cmp("es")
}

func NewEs(cfg *Config, logger jet.CLoggerFunc) (Search, error) {

	s := &esImpl{
		logger: logger,
		cfg:    cfg,
	}
	l := s.l().Mth("new").F(jet.KV{"url": cfg.Url, "sniff": cfg.Sniff})

	opts := []elastic.ClientOptionFunc{elastic.SetURL(s.cfg.Url), elastic.SetSniff(cfg.Sniff)}
	if cfg.Trace {
		opts = append(opts, elastic.SetTraceLog(s.l().Mth("es-trace")))
	}

	// basic auth
	if cfg.Username != "" {
		if cfg.Password == "" {
			return nil, ErrEsBasicAuthInvalid()
		}
		opts = append(opts, elastic.SetBasicAuth(cfg.Username, cfg.Password))
	}
	l.F(jet.KV{"auth": cfg.Username != ""})

	cl, err := elastic.NewClient(opts...)
	if err != nil {
		return nil, ErrEsNewClient(err)
	}
	s.client = cl
	l.Inf("ok")
	return s, nil
}

func (s *esImpl) NewBuilder() IndexBuilder {
	return &esIndexBuilder{
		client: s.client,
		logger: s.logger,
		cfg:    s.cfg,
	}
}

func (s *esImpl) Ping() bool {
	s.l().Mth("ping").Dbg()
	_, code, err := s.client.Ping(s.url).Do(context.Background())
	return err == nil && code == 200
}

func (s *esImpl) Index(ctx context.Context, index string, id string, doc interface{}) error {
	s.l().C(ctx).Mth("indexation").F(jet.KV{"index": index, "id": id}).Dbg().TrcObj("%v", doc)
	svc := s.client.Index().
		Index(index).
		Id(id).
		BodyJson(doc)
	_, err := svc.Do(ctx)
	if err != nil {
		return ErrEsIdx(ctx, err, index, id)
	}
	// refresh
	if s.cfg.Refresh {
		return s.Refresh(ctx, index)
	}
	return nil
}

func (s *esImpl) IndexAsync(ctx context.Context, index string, id string, doc interface{}) {
	goroutine.New().
		WithLogger(s.l().Mth("index-async")).
		Go(ctx, func() {
			l := s.l().C(ctx).Mth("index-async").F(jet.KV{"index": index, "id": id}).Dbg().TrcObj("%v", doc)
			err := s.Index(ctx, index, id, doc)
			if err != nil {
				l.E(err).Err()
			}
		})
}

func (s *esImpl) DeleteIndex(ctx context.Context, index string) error {
	s.l().C(ctx).Mth("delete-index").F(jet.KV{"index": index}).Dbg()
	if exists, err := s.client.IndexExists(index).Do(ctx); err != nil {
		return err
	} else if !exists {
		// nothing to do
		return nil
	}

	if _, err := s.client.DeleteIndex(index).Do(ctx); err != nil {
		return ErrEsDeleteIdx(ctx, err, index)
	}

	return nil
}

func (s *esImpl) IndexBulk(ctx context.Context, index string, docs map[string]interface{}) error {
	s.l().C(ctx).Mth("bulk-indexation").F(jet.KV{"index": index, "docs": len(docs)}).Dbg()
	bulk := s.client.Bulk().Index(index)
	for id, doc := range docs {
		bulk.Add(elastic.NewBulkIndexRequest().Id(id).Doc(doc))
	}
	_, err := bulk.Do(ctx)
	if err != nil {
		return ErrEsBulkIdx(ctx, err, index)
	}
	// refresh
	if s.cfg.Refresh {
		return s.Refresh(ctx, index)
	}
	return nil
}

func (s *esImpl) IndexBulkAsync(ctx context.Context, index string, docs map[string]interface{}) {
	goroutine.New().
		WithLogger(s.l().Mth("index-bulk-async")).
		Go(ctx, func() {
			l := s.l().C(ctx).Mth("bulk-indexation-async").F(jet.KV{"index": index, "docs": len(docs)}).Dbg()
			err := s.IndexBulk(ctx, index, docs)
			if err != nil {
				l.E(err).Err()
			}
		})
}

// Exists checks if a document exists in the index
func (s *esImpl) Exists(ctx context.Context, index, id string) (bool, error) {
	l := s.l().C(ctx).Mth("exists").F(jet.KV{"index": index, "id": id})
	res, err := s.client.Exists().Index(index).Id(id).Do(ctx)
	if err != nil {
		return false, ErrEsExists(ctx, err, index, id)
	}
	l.DbgF("res: %v", res)
	return res, nil
}

func (s *esImpl) Delete(ctx context.Context, index string, id string) error {
	s.l().C(ctx).Mth("delete").F(jet.KV{"index": index, "id": id}).Dbg()
	svc := s.client.
		Delete().
		Index(index).
		Id(id)
	_, err := svc.Do(ctx)
	if err != nil {
		return ErrEsDel(ctx, err, index, id)
	}
	// refresh
	if s.cfg.Refresh {
		return s.Refresh(ctx, index)
	}
	return nil
}

func (s *esImpl) DeleteBulk(ctx context.Context, index string, ids []string) error {
	s.l().C(ctx).Mth("bulk-deletion").F(jet.KV{"index": index, "ids": len(ids)}).Dbg()
	bulk := s.client.Bulk().Index(index)
	for _, id := range ids {
		bulk.Add(elastic.NewBulkDeleteRequest().Id(id))
	}
	_, err := bulk.Do(ctx)
	if err != nil {
		return ErrEsBulkDel(ctx, err, index)
	}
	// refresh
	if s.cfg.Refresh {
		return s.Refresh(ctx, index)
	}
	return nil
}

func (s *esImpl) GetClient() *elastic.Client {
	return s.client
}

func (s *esImpl) Close(ctx context.Context) {
	s.client.Stop()
}

func (s *esImpl) Refresh(ctx context.Context, index string) error {
	_, err := s.client.Refresh(index).Do(ctx)
	if err != nil {
		return ErrEsRefresh(ctx, err, index)
	}
	return nil
}
