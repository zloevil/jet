package jet

import (
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

const (
	ErrCodeCfgRead         = "CFG-001"
	ErrCodeCfgMerge        = "CFG-002"
	ErrCodeCfgUnmarshal    = "CFG-003"
	ErrCodeNoPathSpecified = "CFG-004"
	ErrCodeConfigNotLoaded = "CFG-005"
)

var (
	ErrCfgRead = func(cause error, path string) error {
		return NewAppErrBuilder(ErrCodeCfgRead, "read config error: %s", path).Wrap(cause).Err()
	}
	ErrCfgMerge = func(cause error, path string) error {
		return NewAppErrBuilder(ErrCodeCfgMerge, "merge config error: %s", path).Wrap(cause).Err()
	}
	ErrCfgUnmarshal = func(cause error) error {
		return NewAppErrBuilder(ErrCodeCfgUnmarshal, "merge config error").Wrap(cause).Err()
	}
	ErrNoPathSpecified = func() error {
		return NewAppErrBuilder(ErrCodeNoPathSpecified, "neither env nor explicit path is specified").Business().Err()
	}
	ErrConfigNotLoaded = func() error {
		return NewAppErrBuilder(ErrCodeConfigNotLoaded, "config not loaded").Business().Err()
	}
)

type ConfigLoader[T any] struct {
	paths   []string
	pathEnv string
	prefix  string
}

func NewConfigLoader[T any]() *ConfigLoader[T] {
	return &ConfigLoader[T]{}
}

func (c *ConfigLoader[T]) WithPath(path string) *ConfigLoader[T] {
	c.paths = append(c.paths, path)
	return c
}

func (c *ConfigLoader[T]) WithEnv(env string) *ConfigLoader[T] {
	c.pathEnv = env
	return c
}

func (c *ConfigLoader[T]) WithPrefix(prefix string) *ConfigLoader[T] {
	c.prefix = prefix
	return c
}

// Load loads config by provided parameters
func (c *ConfigLoader[T]) Load() (*T, error) {

	// check params
	if len(c.paths) == 0 && c.pathEnv == "" {
		return nil, ErrNoPathSpecified()
	}

	var res *T

	vpr := viper.New()
	vpr.SetConfigType("yml")
	if c.prefix != "" {
		vpr.SetEnvPrefix(c.prefix)
	}
	vpr.AutomaticEnv()
	vpr.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for i, p := range c.paths {
		absPath, _ := filepath.Abs(p)
		vpr.SetConfigFile(absPath)
		if i == 0 {
			if err := vpr.ReadInConfig(); err != nil {
				return nil, ErrCfgRead(err, absPath)
			}
		} else {
			if err := vpr.MergeInConfig(); err != nil {
				return nil, ErrCfgMerge(err, absPath)
			}
		}
	}

	// get path from ENV only if explicit paths hasn't been specified
	if len(c.paths) == 0 && c.pathEnv != "" {
		path := os.Getenv(c.pathEnv)
		if path != "" {
			absPath, _ := filepath.Abs(path)
			vpr.SetConfigFile(absPath)
			if err := vpr.ReadInConfig(); err != nil {
				return nil, ErrCfgRead(err, absPath)
			}
		}
	}

	if len(vpr.AllKeys()) == 0 {
		return nil, ErrConfigNotLoaded()
	}

	if err := vpr.Unmarshal(&res); err != nil {
		return nil, ErrCfgUnmarshal(err)
	}

	return res, nil
}
