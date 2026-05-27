package jet

import (
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type configTestSuite struct {
	Suite
}

func (s *configTestSuite) SetupSuite() {
	s.Suite.Init(func() CLogger { return L(InitLogger(&LogConfig{Level: TraceLevel})) })
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(configTestSuite))
}

type TestCfg struct {
	Some struct {
		Attr            string
		VeryComplexName string `mapstructure:"very_complex_name"`
	}
}

func (s *configTestSuite) Test_WhenExplicitPath_EnvDoesntWork() {
	loader := NewConfigLoader[TestCfg]()
	cfg, err := loader.WithPath("config_test.yml").WithEnv("not-existent-path").Load()
	s.NoError(err)
	s.NotEmpty(cfg)
	s.NotEmpty(cfg.Some)
	s.NotEmpty(cfg.Some.Attr)
	s.NotEmpty(cfg.Some.VeryComplexName)
}

func (s *configTestSuite) Test_WhenEnv() {
	_ = os.Setenv("TEST_CONFIG_PATH", "config_test.yml")
	loader := NewConfigLoader[TestCfg]()
	cfg, err := loader.WithEnv("TEST_CONFIG_PATH").Load()
	s.NoError(err)
	s.NotEmpty(cfg)
	s.NotEmpty(cfg.Some)
	s.NotEmpty(cfg.Some.Attr)
	s.NotEmpty(cfg.Some.VeryComplexName)
}

func (s *configTestSuite) Test_WhenNoValidPath() {
	_ = os.Setenv("TEST_CONFIG_PATH_INVALID", "")
	loader := NewConfigLoader[TestCfg]()
	_, err := loader.WithEnv("TEST_CONFIG_PATH_INVALID").Load()
	s.AssertAppErr(err, ErrCodeConfigNotLoaded)
}

func (s *configTestSuite) Test_WhenEnv_ComplexName() {
	_ = os.Setenv("SOME_VERY_COMPLEX_NAME", "value")
	loader := NewConfigLoader[TestCfg]()
	cfg, err := loader.WithPath("config_test.yml").Load()
	s.NoError(err)
	s.NotEmpty(cfg)
	s.NotEmpty(cfg.Some)
	s.NotEmpty(cfg.Some.Attr)
	s.Equal("value", cfg.Some.VeryComplexName)
}
