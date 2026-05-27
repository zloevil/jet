package elasticsearch

import (
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
)

type esModelMappingTestSuite struct {
	jet.Suite
}

func (s *esModelMappingTestSuite) SetupSuite() {
	s.Suite.Init(func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) })
}

func (s *esModelMappingTestSuite) SetupTest() {
}

func (s *esModelMappingTestSuite) TearDownSuite() {}

func TestEsModelMappingSuite(t *testing.T) {
	suite.Run(t, new(esModelMappingTestSuite))
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenNil() {
	b := &esIndexBuilder{logger: s.L}
	mapping, _ := b.modelToMapping(s.Ctx, nil)
	s.Empty(mapping)
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenNoJsonMapping() {
	b := &esIndexBuilder{logger: s.L}
	model := &struct {
		Field string `es:"type:keyword"`
	}{
		Field: "value",
	}
	_, err := b.modelToMapping(s.Ctx, model)
	s.Error(err)
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenJsonEmptyField() {
	b := &esIndexBuilder{logger: s.L}
	model := &struct {
		Field string `json:"" es:"type:keyword"`
	}{
		Field: "value",
	}
	_, err := b.modelToMapping(s.Ctx, model)
	s.Error(err)
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenNotPointerType() {
	b := &esIndexBuilder{logger: s.L}
	model := struct {
		Field string `json:"" es:"type:keyword"`
	}{
		Field: "value",
	}
	_, err := b.modelToMapping(s.Ctx, model)
	s.Error(err)
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenNotStructType() {
	b := &esIndexBuilder{logger: s.L}
	model := "string"
	_, err := b.modelToMapping(s.Ctx, model)
	s.Error(err)
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenWithType() {
	b := &esIndexBuilder{logger: s.L}
	model := &struct {
		Field string `json:"field" es:"type:keyword"`
	}{
		Field: "value",
	}
	mapping, err := b.modelToMapping(s.Ctx, model)
	s.NoError(err)
	s.NotEmpty(mapping)
	s.Equal(1, len(mapping.Mappings.Properties))
	f, ok := mapping.Mappings.Properties["field"]
	s.True(ok)
	s.Equal("keyword", f.Type)
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenNoEsTag_Skipped() {
	b := &esIndexBuilder{logger: s.L}
	model := &struct {
		Field1 string `json:"field1"`
		Field2 string `json:"field2" es:"type:text"`
	}{
		Field1: "value",
		Field2: "value",
	}
	mapping, err := b.modelToMapping(s.Ctx, model)
	s.NoError(err)
	s.NotEmpty(mapping)
	s.Equal(1, len(mapping.Mappings.Properties))
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenEsTagEmpty_Skipped() {
	b := &esIndexBuilder{logger: s.L}
	model := &struct {
		Field1 string `json:"field1" es:""`
		Field2 string `json:"field2" es:"type:text"`
	}{
		Field1: "value",
		Field2: "value",
	}
	mapping, err := b.modelToMapping(s.Ctx, model)
	s.NoError(err)
	s.NotEmpty(mapping)
	s.Equal(1, len(mapping.Mappings.Properties))
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenNoOneEsTag() {
	b := &esIndexBuilder{logger: s.L}
	model := &struct {
		Field string `json:"field"`
	}{
		Field: "value",
	}
	_, err := b.modelToMapping(s.Ctx, model)
	s.Error(err)
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenTypeWrong() {
	b := &esIndexBuilder{logger: s.L}
	model := &struct {
		Field string `json:"field" es:"type:wrong"`
	}{
		Field: "value",
	}
	_, err := b.modelToMapping(s.Ctx, model)
	s.Error(err)
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WhenTagWrong() {
	b := &esIndexBuilder{logger: s.L}
	model := &struct {
		Field string `json:"field es:wrong"`
	}{
		Field: "value",
	}
	_, err := b.modelToMapping(s.Ctx, model)
	s.Error(err)
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WithTwoFields() {
	b := &esIndexBuilder{logger: s.L}
	model := &struct {
		Field1 string `json:"field1" es:"type:keyword"`
		Field2 string `json:"field2" es:"type:text"`
	}{
		Field1: "value",
		Field2: "value",
	}
	mapping, err := b.modelToMapping(s.Ctx, model)
	s.NoError(err)
	s.NotEmpty(mapping)
	s.Equal(2, len(mapping.Mappings.Properties))
	f, ok := mapping.Mappings.Properties["field1"]
	s.True(ok)
	s.Equal("keyword", f.Type)
	f, ok = mapping.Mappings.Properties["field2"]
	s.True(ok)
	s.Equal("text", f.Type)
}

func (s *esModelMappingTestSuite) Test_ModelToMapping_WithNoIndex() {
	b := &esIndexBuilder{logger: s.L}
	model := &struct {
		Field1 string `json:"field1" es:"type:keyword"`
		Field2 string `json:"field2" es:"-"`
		Field3 string `json:"field3" es:"type:text;-"`
	}{
		Field1: "value",
		Field2: "value",
		Field3: "value",
	}
	mapping, err := b.modelToMapping(s.Ctx, model)
	s.NoError(err)
	s.NotEmpty(mapping)
	s.Equal(3, len(mapping.Mappings.Properties))
	f, ok := mapping.Mappings.Properties["field1"]
	s.True(ok)
	s.Equal("keyword", f.Type)
	f, ok = mapping.Mappings.Properties["field2"]
	s.True(ok)
	s.Equal("text", f.Type)
	s.False(*f.Index)
	f, ok = mapping.Mappings.Properties["field3"]
	s.True(ok)
	s.Equal("text", f.Type)
	s.False(*f.Index)
}
