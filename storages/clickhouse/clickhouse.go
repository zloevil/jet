package clickhouse

import (
	"database/sql"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/zloevil/jet"
)

type ClickHouse struct {
	Instance clickhouse.Conn
	cfg      *Config
	logger   jet.CLoggerFunc
}

type Engines struct {
	Kafka map[string]*KafkaEngine
}

type KafkaEngine struct {
	BrokerList   string `mapstructure:"broker_list"`
	TopicList    string `mapstructure:"topic_list"`
	GroupName    string `mapstructure:"group_name"`
	NumConsumers uint   `mapstructure:"num_consumers"`
}

// Config configuration parameters
type Config struct {
	User     string // User username
	Password string // Password password
	Database string // Database database name
	Port     string // Port connection
	Host     string // Host connection
	Debug    bool   // Debug if debug mode enabled
	Engines  *Engines
}

func Open(config *Config, logger jet.CLoggerFunc) (*ClickHouse, error) {
	s := &ClickHouse{
		logger: logger,
		cfg:    config,
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.User,
			Password: config.Password,
		},
		Debug: config.Debug,
		Debugf: func(format string, v ...interface{}) {
			logger().Cmp("click").Mth("debug").DbgF(format, v...)
		},
	})
	if err != nil {
		return nil, ErrClickOpen(err)
	}
	s.Instance = conn
	v, err := conn.ServerVersion()
	if err != nil {
		return nil, ErrClickGetVer(err)
	}
	logger().Pr("click").Cmp(config.User).Mth("open").F(jet.KV{"version": v}).Inf("ok")
	return s, nil
}

func OpenDb(config *Config, logger jet.CLoggerFunc) (*sql.DB, error) {

	// make connection
	conn := clickhouse.OpenDB(cfgToOptions(config, logger))

	// ping
	err := conn.Ping()
	if err != nil {
		return nil, ErrClickPing(err)
	}

	return conn, nil
}

func (s *ClickHouse) l() jet.CLogger {
	return s.logger().Cmp("click")
}

func (s *ClickHouse) Close() {
	if s.Instance != nil {
		_ = s.Instance.Close()
		s.Instance = nil
	}
	s.logger().Cmp("click").Mth("close").Inf("ok")
}

func cfgToOptions(config *Config, logger jet.CLoggerFunc) *clickhouse.Options {
	return &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.User,
			Password: config.Password,
		},
		Debug: config.Debug,
		Debugf: func(format string, v ...interface{}) {
			logger().Cmp("click").Mth("debug").DbgF(format, v...)
		},
	}
}
