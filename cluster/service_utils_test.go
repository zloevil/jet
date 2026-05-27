package cluster

import (
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"testing"
)

type serviceUtilsTestSuite struct {
	jet.Suite
}

func (s *serviceUtilsTestSuite) SetupSuite() {
	s.Suite.Init(nil)
}

func TestServiceUtilsSuite(t *testing.T) {
	suite.Run(t, new(serviceUtilsTestSuite))
}

func (s *serviceUtilsTestSuite) Test_GetServiceRootPath_WhenEmptyInput() {
	s.Empty(GetServiceRootPath(""))
}

func (s *serviceUtilsTestSuite) Test_GetServiceRootPath_WhenNotExistent() {
	s.Empty(GetServiceRootPath("some"))
}

func (s *serviceUtilsTestSuite) Test_GetServiceRootPath_WhenExists() {
	// this test file lives in the "cluster" directory, so walking up from it
	// always finds an ancestor named "cluster" regardless of the checkout location
	s.NotEmpty(GetServiceRootPath("cluster"))
}
