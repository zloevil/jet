package migration

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/zloevil/jet"
	"os"
	"path/filepath"
)

const (
	pgMigrationAdvisoryLockId = 789654123

	DialectPostgres   = "postgres"
	DialectClickHouse = "clickhouse"
)

// Migration allows applying database migrations
type Migration interface {
	// Up applies all migrations up to the final version
	Up() error
	// Down rollbacks single version
	Down() error
}

type migImpl struct {
	db      *sql.DB
	source  string
	logger  jet.CLoggerFunc
	dialect string
}

func NewMigration(db *sql.DB, source string, logger jet.CLoggerFunc, dialect string) Migration {
	return &migImpl{
		db:      db,
		source:  source,
		logger:  logger,
		dialect: dialect,
	}
}

func (m *migImpl) l() jet.CLogger {
	return m.logger().Cmp("db-mig")
}

func (m *migImpl) Up() error {
	l := m.l().Mth("up").InfF("applying from %s ...", m.source)
	return m.exec(func() error {
		if err := goose.Up(m.db, m.source); err != nil {
			return ErrGooseMigrationUp(err)
		}
		return nil
	}, l)
}

func (m *migImpl) Down() error {
	l := m.l().Mth("down").InfF("applying from %s ...", m.source)
	return m.exec(func() error {
		if err := goose.Down(m.db, m.source); err != nil {
			return ErrGooseMigrationDown(err)
		}
		return nil
	}, l)
}

func (m *migImpl) exec(fn func() error, l jet.CLogger) error {

	// check dialect
	if m.dialect != DialectPostgres && m.dialect != DialectClickHouse {
		return ErrGooseUnsupportedDialect(m.dialect)
	}

	// check folder exists
	absPath, _ := filepath.Abs(m.source)
	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return ErrGooseFolderNotFound(absPath)
		}
		return ErrGooseFolderOpen(err)
	}

	// ping database
	err := m.db.Ping()
	if err != nil {
		return ErGoosePing(err)
	}

	// lock is currently supported for Postgres only
	if m.dialect == DialectPostgres {

		// lock before migration (applying advisory lock) to guaranty exclusive migration execution
		_, err := m.db.Exec(fmt.Sprintf("select pg_advisory_lock(%d)", pgMigrationAdvisoryLockId))
		if err != nil {
			l.E(ErrGooseMigrationLock(err)).Err()
		}

		// unlock after migration
		defer func() {
			if _, err := m.db.Exec(fmt.Sprintf("select pg_advisory_unlock(%d)", pgMigrationAdvisoryLockId)); err != nil {
				m.logger().E(ErrGooseMigrationUnLock(err)).Err()
			}
		}()
	}

	// set dialect
	_ = goose.SetDialect(m.dialect)

	// execute migration action
	err = fn()
	if err != nil {
		return err
	}

	// get current version
	version, err := goose.GetDBVersion(m.db)
	if err != nil {
		return ErrGooseMigrationGetVer(err)
	}

	l.InfF("ok, version: %d", version)
	return nil
}
