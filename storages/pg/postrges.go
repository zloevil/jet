package pg

import (
	"fmt"
	"github.com/zloevil/jet"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"time"
)

type Storage struct {
	Instance *gorm.DB
	DBName   string
	logger   jet.CLoggerFunc
}

// DbConfig database configuration
type DbConfig struct {
	ConnectionString string `mapstructure:"connection_string"` // ConnectionString if specified overrides all other connection params
	User             string
	Password         string
	DBName           string
	Port             string
	Host             string
}

// DbClusterConfig configuration of database cluster
type DbClusterConfig struct {
	Master *DbConfig // Master database
	Slave  *DbConfig // Slave database
}

func Open(config *DbConfig, logger jet.CLoggerFunc) (*Storage, error) {

	s := &Storage{
		DBName: config.DBName,
		logger: logger,
	}

	dsn := config.ConnectionString
	if dsn == "" {
		dsn = fmt.Sprintf("user=%s password=%s dbname=%s port=%s host=%s",
			config.User,
			config.Password,
			config.DBName,
			config.Port,
			config.Host,
		)
	}

	// uncomment to log all queries
	cfg := &gorm.Config{
		Logger: gormLogger.New(
			logger(),
			//log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
			gormLogger.Config{
				SlowThreshold: time.Second * 10, // Slow SQL threshold
				LogLevel:      gormLogger.Info,  // Log level
				Colorful:      false,            // Disable color
			},
		),
		NowFunc: func() time.Time { return jet.Now() },
	}

	db, err := gorm.Open(postgres.Open(dsn), cfg)
	if err != nil {
		return nil, ErrPostgresOpen(err)
	}

	logger().Pr("db").Cmp(config.User).Inf("ok")

	s.Instance = db

	return s, nil
}

func (s *Storage) Close() {
	if s.Instance != nil {
		db, _ := s.Instance.DB()
		_ = db.Close()
	}
}
