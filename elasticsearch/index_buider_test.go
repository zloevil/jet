//go:build example

package elasticsearch

import (
	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
)

type esIndexTestSuite struct {
	jet.Suite
}

func (s *esIndexTestSuite) SetupSuite() {
	s.Suite.Init(func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) })
}

func (s *esIndexTestSuite) SetupTest() {
}

func (s *esIndexTestSuite) TearDownSuite() {}

func TestEsIndexSuite(t *testing.T) {
	suite.Run(t, new(esIndexTestSuite))
}

const (
	url = "http://localhost:9200"
)

func (s *esIndexTestSuite) Test_Index_ChangingModelMappingWithCfgSettings() {
	es, err := NewEs(&Config{
		Url:      url,
		Trace:    true,
		Sniff:    true,
		Shards:   2,
		Replicas: 2,
	}, s.L)
	s.NoError(err)

	index := jet.NewRandString()
	type model struct {
		Field string `json:"field" es:"type:keyword"`
	}
	err = es.NewBuilder().WithIndex(index).WithMappingModel(&model{}).Build(s.Ctx)
	s.NoError(err)

	type modelNew struct {
		Field  string `json:"field" es:"type:keyword"`
		Field2 string `json:"field2" es:"type:keyword"`
	}
	err = es.NewBuilder().WithIndex(index).WithMappingModel(&modelNew{}).Build(s.Ctx)
	s.NoError(err)
}

func (s *esIndexTestSuite) Test_Index_ChangingModelMapping() {
	es, err := NewEs(&Config{
		Url:   url,
		Trace: false,
		Sniff: true,
	}, s.L)
	s.NoError(err)

	type model struct {
		Field string `json:"field" es:"type:keyword"`
	}
	index := jet.NewRandString()
	err = es.NewBuilder().WithIndex(index).WithMappingModel(&model{}).Build(s.Ctx)
	s.NoError(err)

	type modelNew struct {
		Field  string `json:"field" es:"type:keyword"`
		Field2 string `json:"field2" es:"type:keyword"`
	}

	err = es.NewBuilder().WithIndex(index).WithMappingModel(&modelNew{}).Build(s.Ctx)
	s.NoError(err)
}

func (s *esIndexTestSuite) Test_Index_ChangingMapping_ExistentFields() {
	es, err := NewEs(&Config{
		Url:   url,
		Trace: false,
		Sniff: true,
	}, s.L)
	s.NoError(err)

	type model struct {
		Field string `json:"field" es:"type:keyword"`
	}

	index := jet.NewRandString()
	err = es.NewBuilder().WithIndex(index).WithMappingModel(&model{}).Build(s.Ctx)
	s.NoError(err)
	type modelNew struct {
		Field string `json:"field" es:"type:text"`
	}
	err = es.NewBuilder().WithIndex(index).WithMappingModel(&modelNew{}).Build(s.Ctx)
	s.NotNil(err)
}

func (s *esIndexTestSuite) Test_Index_ChangingMapping_ExplicitMapping() {
	es, err := NewEs(&Config{
		Url:      url,
		Trace:    true,
		Sniff:    true,
		Shards:   2,
		Replicas: 2,
	}, s.L)
	s.NoError(err)

	mapping := `
{
	"mappings": {
		"properties": {
			"field1": {
				"type": "keyword"
			}
		}
	}
}
`
	index := jet.NewRandString()
	err = es.NewBuilder().WithIndex(index).WithExplicitMapping(mapping).Build(s.Ctx)
	s.NoError(err)
	newMapping := `
{
	"mappings": {
		"properties": {
			"field1": {
				"type": "keyword"
			},
			"field2": {
				"type": "keyword"
			}
		}
	}
}
`
	err = es.NewBuilder().WithIndex(index).WithExplicitMapping(newMapping).Build(s.Ctx)
	s.NoError(err)
}

func (s *esIndexTestSuite) Test_Index_Mapping_WhenNotIndexField() {
	es, err := NewEs(&Config{
		Url:   url,
		Trace: false,
		Sniff: true,
	}, s.L)
	s.NoError(err)

	type model struct {
		Field1 string `json:"field1" es:"type:keyword"`
		Field2 string `json:"field2" es:"type:keyword;-"`
	}

	index := jet.NewRandString()
	err = es.NewBuilder().WithIndex(index).WithMappingModel(&model{}).Build(s.Ctx)
	s.NoError(err)
}

func (s *esIndexTestSuite) Test_Alias_ChangingModelMapping() {
	es, err := NewEs(&Config{
		Url:      url,
		Trace:    true,
		Sniff:    true,
		Shards:   2,
		Replicas: 2,
	}, s.L)
	s.NoError(err)

	// create alias and index
	alias := jet.NewRandString()
	type model struct {
		Field string `json:"field" es:"type:keyword"`
	}
	err = es.NewBuilder().WithAlias(alias).WithMappingModel(&model{}).Build(s.Ctx)
	s.NoError(err)

	// index document
	err = es.Index(s.Ctx, alias, jet.NewId(), &model{Field: "value"})
	s.NoError(err)

	// search
	err = es.Refresh(s.Ctx, alias)
	s.NoError(err)

	// get data from alias
	srchRs, err := es.GetClient().Search().Index(alias).Query(elastic.NewMatchAllQuery()).Do(s.Ctx)
	s.NoError(err)
	s.Equal(int64(1), srchRs.TotalHits())

	// change mapping
	type modelNew struct {
		Field  string `json:"field" es:"type:keyword"`
		Field2 string `json:"field2" es:"type:keyword"`
	}
	err = es.NewBuilder().WithAlias(alias).WithMappingModel(&modelNew{}).Build(s.Ctx)
	s.NoError(err)

	// index document
	err = es.Index(s.Ctx, alias, jet.NewId(), &modelNew{Field: "value", Field2: "value2"})
	s.NoError(err)

	// search
	err = es.Refresh(s.Ctx, alias)
	s.NoError(err)

	// get data from alias
	srchRs, err = es.GetClient().Search().Index(alias).Query(elastic.NewMatchAllQuery()).Do(s.Ctx)
	s.NoError(err)
	s.Equal(int64(2), srchRs.TotalHits())
}
