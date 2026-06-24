package cluster

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/storages/clickhouse"
	"github.com/zloevil/jet/storages/migration"
	jetStorage "github.com/zloevil/jet/storages/pg"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
)

// Bootstrap interface needs to be implemented by service instance implementation to handle service's lifecycle
type Bootstrap interface {
	// Init initializes the service
	Init(ctx context.Context, cfg any) error
	// Start executes all background processes
	Start(ctx context.Context) error
	// Close closes the service
	Close(ctx context.Context)
}

const (
	configFlagName                        = "config"
	configDefaultPath                     = "./config.yml"
	migrationSourceFlagName               = "source"
	migrationSourceDefaultPath            = "./db/migrations"
	defaultSchema                         = "default"
	scriptParameterizedWithConfigFlagName = "parameterized"
	tmpMigrationDirPrefix                 = "tmp"

	ErrCodeConfigPathNotSpecified      = "SVS-001"
	ErrCodeMigrationSourcePathInvalid  = "SVS-002"
	ErrCodeMigrationSourceParamInvalid = "SVS-003"
	ErrCodeClickConfigInvalid          = "SVS-004"
	ErrCodePgConfigInvalid             = "SVS-005"
	ErrCodeApplyConfigReadDirFailed    = "SVS-006"
	ErrCodeApplyConfigMkdirTempFailed  = "SVS-007"
	ErrCodeApplyConfigReadFileFailed   = "SVS-008"
	ErrCodeApplyConfigWriteFileFailed  = "SVS-009"
)

var (
	ErrConfigPathNotSpecified = func() error {
		return jet.NewAppErrBuilder(ErrCodeConfigPathNotSpecified, "").Business().Err()
	}
	ErrMigrationSourcePathInvalid = func() error {
		return jet.NewAppErrBuilder(ErrCodeMigrationSourcePathInvalid, "migration source path isn't valid").Business().Err()
	}
	ErrMigrationSourceParamInvalid = func() error {
		return jet.NewAppErrBuilder(ErrCodeMigrationSourceParamInvalid, "command param isn't valid").Business().Err()
	}
	ErrClickConfigInvalid = func() error {
		return jet.NewAppErrBuilder(ErrCodeClickConfigInvalid, "click config isn't valid").Business().Err()
	}
	ErrPgConfigInvalid = func() error {
		return jet.NewAppErrBuilder(ErrCodePgConfigInvalid, "pg config isn't valid").Business().Err()
	}
	ErrApplyConfigReadDirFailed = func(err error) error {
		return jet.NewAppErrBuilder(ErrCodeApplyConfigReadDirFailed, "apply config reading directory failed").Wrap(err).Err()
	}
	ErrApplyConfigMkdirTempFailed = func(err error) error {
		return jet.NewAppErrBuilder(ErrCodeApplyConfigMkdirTempFailed, "apply config making directory failed").Wrap(err).Err()
	}
	ErrApplyConfigReadFileFailed = func(err error) error {
		return jet.NewAppErrBuilder(ErrCodeApplyConfigReadFileFailed, "apply config reading file failed").Wrap(err).Err()
	}
	ErrApplyConfigWriteFileFailed = func(err error) error {
		return jet.NewAppErrBuilder(ErrCodeApplyConfigWriteFileFailed, "apply config writing file failed").Wrap(err).Err()
	}
)

type ServiceInstance[TCfg any] struct {
	svcCode      string         // svcCode unique identifier of the service
	nodeId       string         // nodeId service node ID
	instanceId   string         // instanceId service instance when multiple replicas are running
	bootstrap    Bootstrap      // bootstrap implementation of Bootstrap interface
	rootCmd      *cobra.Command // rootCmd root command
	confPathEnv  *string        // confPathEnv env var name of config path (optional)
	migSourceEnv *string        // migSourceEnv env var name of migration path (optional)
	logger       *jet.Logger
}

func New[TCfg any](svcCode string, bootstrap Bootstrap) *ServiceInstance[TCfg] {

	s := &ServiceInstance[TCfg]{
		bootstrap: bootstrap,
		svcCode:   svcCode,
		logger:    jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel, Format: jet.FormatterJson}),
	}

	// init root command
	s.rootCmd = &cobra.Command{
		Use: svcCode,
	}
	flags := s.rootCmd.PersistentFlags()
	flags.String(
		configFlagName,
		configDefaultPath,
		"--config <path-to-file>",
	)

	// app command
	appCmd := &cobra.Command{
		Use: "app",
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.executeAppCmd(cmd, args)
		},
	}
	s.rootCmd.AddCommand(appCmd)

	return s
}

func (s *ServiceInstance[TCfg]) l() jet.CLogger {
	return s.GetLogger()()
}

// WithConfigPathEnv allows to specify env which is used to obtain a config path. It has priority over command flags
func (s *ServiceInstance[TCfg]) WithConfigPathEnv(env string) *ServiceInstance[TCfg] {
	if env != "" {
		s.confPathEnv = &env
	}
	return s
}

// WithMigrationSourceEnv allows to specify env which is used to obtain a migration source folder path. It has priority over command flags
func (s *ServiceInstance[TCfg]) WithMigrationSourceEnv(env string) *ServiceInstance[TCfg] {
	if env != "" {
		s.migSourceEnv = &env
	}
	return s
}

func (s *ServiceInstance[TCfg]) WithDbMigration(getDbConfigFn func(cfg *TCfg) (any, error)) *ServiceInstance[TCfg] {

	// create migration commands
	dbUpCmd := &cobra.Command{
		Use: "db-up",
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.executePgCmd(cmd, getDbConfigFn, true)
		},
	}
	dbDownCmd := &cobra.Command{
		Use: "db-down",
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.executePgCmd(cmd, getDbConfigFn, false)
		},
	}

	s.rootCmd.AddCommand(dbUpCmd, dbDownCmd)

	// set flags
	flags := dbUpCmd.PersistentFlags()
	flags.String(
		migrationSourceFlagName,
		migrationSourceDefaultPath,
		"--source <path-to-migration-folder>",
	)

	flags = dbDownCmd.PersistentFlags()
	flags.String(
		migrationSourceFlagName,
		migrationSourceDefaultPath,
		"--source <path-to-migration-folder>",
	)

	return s
}

func (s *ServiceInstance[TCfg]) WithClickHouseMigration(getClickConfigFn func(cfg *TCfg) (any, error)) *ServiceInstance[TCfg] {

	var upSources []string
	var downSources []string

	// create migration commands
	dbUpCmd := &cobra.Command{
		Use: "ch-up",
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.executeClickCmd(cmd, getClickConfigFn, true)
		},
	}
	dbDownCmd := &cobra.Command{
		Use: "ch-down",
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.executeClickCmd(cmd, getClickConfigFn, false)
		},
	}

	s.rootCmd.AddCommand(dbUpCmd, dbDownCmd)

	// set flags
	flags := dbUpCmd.PersistentFlags()
	flags.StringArrayVar(
		&upSources,
		migrationSourceFlagName,
		[]string{migrationSourceDefaultPath},
		"--source <path-to-migration-folder>",
	)
	flags.String(
		scriptParameterizedWithConfigFlagName,
		"false",
		"--parameterized <true/false>",
	)

	flags = dbDownCmd.PersistentFlags()
	flags.StringArrayVar(
		&downSources,
		migrationSourceFlagName,
		[]string{migrationSourceDefaultPath},
		"--source <path-to-migration-folder>",
	)

	return s
}

func (s *ServiceInstance[TCfg]) WithClickHouseMigrations(getClickConfigFns ...func(cfg *TCfg) (any, error)) *ServiceInstance[TCfg] {

	var upSources []string
	var downSources []string

	// create migration commands
	dbUpCmd := &cobra.Command{
		Use: "ch-up",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, getCfgFn := range getClickConfigFns {
				if err := s.executeClickCmd(cmd, getCfgFn, true); err != nil {
					return err
				}
			}
			return nil
		},
	}
	dbDownCmd := &cobra.Command{
		Use: "ch-down",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, getCfgFn := range getClickConfigFns {
				if err := s.executeClickCmd(cmd, getCfgFn, false); err != nil {
					return err
				}
			}
			return nil
		},
	}

	s.rootCmd.AddCommand(dbUpCmd, dbDownCmd)

	// set flags
	flags := dbUpCmd.PersistentFlags()
	flags.StringArrayVar(
		&upSources,
		migrationSourceFlagName,
		[]string{},
		"--source <user1>:<path-to-migration-folder1> --source <user2>:<path-to-migration-folder2>",
	)
	flags.String(
		scriptParameterizedWithConfigFlagName,
		"false",
		"--parameterized <true/false>",
	)

	flags = dbDownCmd.PersistentFlags()
	flags.StringArrayVar(
		&downSources,
		migrationSourceFlagName,
		[]string{},
		"--source <user1>:<path-to-migration-folder1> --source <user2>:<path-to-migration-folder2>",
	)

	return s
}

func (s *ServiceInstance[TCfg]) GetCode() string {
	return s.svcCode
}

func (s *ServiceInstance[TCfg]) NodeId() string {
	return s.nodeId
}

func (s *ServiceInstance[TCfg]) Execute() error {
	return s.rootCmd.Execute()
}

func (s *ServiceInstance[TCfg]) GetLogger() jet.CLoggerFunc {
	return func() jet.CLogger {
		return jet.L(s.logger).Srv(s.svcCode).Node(s.nodeId)
	}
}

func (s *ServiceInstance[TCfg]) loadConfig(cmd *cobra.Command) (*TCfg, error) {
	l := s.l().Mth("load-config")

	configLoader := jet.NewConfigLoader[TCfg]()
	var path string

	// get from cmd flag
	if f := cmd.Flag(configFlagName); f != nil {
		path = f.Value.String()
		if path != "" {
			configLoader = configLoader.WithPath(path)
		}
	} else if s.confPathEnv != nil {
		// try to load config from the path passed by env
		configLoader = configLoader.WithEnv(*s.confPathEnv)
		path = os.Getenv(*s.confPathEnv)
	} else {
		return nil, ErrConfigPathNotSpecified()
	}

	// try to load from passed path
	absPath, _ := filepath.Abs(path)
	config, err := configLoader.Load()
	if err != nil {
		return nil, err
	}
	if config != nil {
		l.DbgF("found: %s", absPath).TrcObj("%v", config)
	}

	return config, nil
}

func (s *ServiceInstance[TCfg]) absPath(src string) (string, error) {

	if src == "" {
		return "", nil
	}

	v, _ := filepath.Abs(src)
	if v == "" {
		return "", nil
	}

	if _, err := os.Stat(v); err != nil {
		return "", ErrMigrationSourcePathInvalid()
	}

	return v, nil
}

func (s *ServiceInstance[TCfg]) loadMigrationSources(cmd *cobra.Command) (map[string]string, error) {

	res := make(map[string]string)

	// get from cmd flag
	sourcesFlags, _ := cmd.Flags().GetStringArray(migrationSourceFlagName)

	if len(sourcesFlags) == 0 {
		v, _ := cmd.Flags().GetString(migrationSourceFlagName)
		if v != "" {
			sourcesFlags = []string{v}
		}
	}

	parseSourceFn := func(src string) (string, string, error) {

		schemaSource := strings.Split(src, ":")

		switch len(schemaSource) {
		case 1:
			absPath, err := s.absPath(schemaSource[0])
			if err != nil {
				return "", "", err
			}
			return defaultSchema, absPath, nil
		case 2:
			absPath, err := s.absPath(schemaSource[1])
			if err != nil {
				return "", "", err
			}
			return schemaSource[0], absPath, nil
		default:
			return "", "", ErrMigrationSourceParamInvalid()
		}
	}

	for _, source := range sourcesFlags {
		if source == "" {
			continue
		}

		k, v, err := parseSourceFn(source)
		if err != nil {
			return nil, err
		}
		res[k] = v

	}

	// try to load a migration source from the path passed by env
	if len(res) == 0 && s.migSourceEnv != nil {
		source := os.Getenv(*s.migSourceEnv)
		if source != "" {
			for _, src := range strings.Split(source, ",") {
				if src == "" {
					continue
				}

				k, v, err := parseSourceFn(src)
				if err != nil {
					return nil, err
				}
				res[k] = v
			}
		}
	}

	return res, nil
}

func (s *ServiceInstance[TCfg]) executeAppCmd(cmd *cobra.Command, args []string) error {
	l := s.l().Mth("app")

	// load config
	config, err := s.loadConfig(cmd)
	if err != nil {
		return err
	}

	// init context
	ctx, cancelFn := context.WithCancel(jet.NewRequestCtx().Empty().WithNewRequestId().ToContext(context.Background()))
	defer cancelFn()

	// init service
	if err = s.bootstrap.Init(ctx, config); err != nil {
		l.E(err).Err("init: fail")
		return err
	}
	l.Inf("init: ok")

	// start listening
	if err := s.bootstrap.Start(ctx); err != nil {
		l.C(ctx).E(err).Err("start: fail")
		return err
	}
	l.Inf("started: ok")

	// close
	defer func() {
		s.bootstrap.Close(ctx)
		l.Inf("graceful shutdown")
	}()

	// handle app close
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	return nil
}

func (s *ServiceInstance[TCfg]) executePgCmd(cmd *cobra.Command, getDbConfigFn func(cfg *TCfg) (any, error), up bool) error {

	// load config
	config, err := s.loadConfig(cmd)
	if err != nil {
		return err
	}

	// extract config
	dbConfig, err := getDbConfigFn(config)
	if err != nil {
		return err
	}
	pgDbConfig, ok := dbConfig.(*jetStorage.DbConfig)
	if !ok || pgDbConfig == nil {
		return ErrPgConfigInvalid()
	}

	// load migration source
	schemaSources, err := s.loadMigrationSources(cmd)
	if err != nil {
		return err
	}

	var src string
	if len(schemaSources) > 1 {
		src, ok = schemaSources[pgDbConfig.User]
		if !ok {
			return ErrMigrationSourceParamInvalid()
		}
	} else if len(schemaSources) == 1 {
		src = schemaSources[defaultSchema]
	} else {
		return ErrMigrationSourceParamInvalid()
	}

	// build a function opening database
	openDb := func() (*sql.DB, error) {
		pg, err := jetStorage.Open(pgDbConfig, s.GetLogger())
		if err != nil {
			return nil, err
		}
		db, _ := pg.Instance.DB()
		return db, nil
	}

	// we count on that migrations is located either in "src" folder or in "src/pg" folder
	return s.executeMigrationCmd(openDb, migration.DialectPostgres, []string{fmt.Sprintf("%s/pg", src), src}, up)
}

func (s *ServiceInstance[TCfg]) executeClickCmd(cmd *cobra.Command, getClickConfigFn func(cfg *TCfg) (any, error), up bool) error {

	// load config
	config, err := s.loadConfig(cmd)
	if err != nil {
		return err
	}

	// extract config
	dbConfig, err := getClickConfigFn(config)
	if err != nil {
		return err
	}
	clickDbCfg, ok := dbConfig.(*clickhouse.Config)
	if !ok || clickDbCfg == nil {
		return ErrClickConfigInvalid()
	}

	// load migration source
	schemaSources, err := s.loadMigrationSources(cmd)
	if err != nil {
		return err
	}

	var src string
	if len(schemaSources) > 1 {
		src, ok = schemaSources[clickDbCfg.User]
		if !ok {
			return ErrMigrationSourceParamInvalid()
		}
	} else if len(schemaSources) == 1 {
		src = schemaSources[defaultSchema]
	} else {
		return ErrMigrationSourceParamInvalid()
	}

	// build a function opening database
	openDb := func() (*sql.DB, error) {
		db, err := clickhouse.OpenDb(clickDbCfg, s.GetLogger())
		if err != nil {
			return nil, err
		}
		return db, nil
	}

	var paths = []string{fmt.Sprintf("%s/click", src), src}
	if f := cmd.Flag(scriptParameterizedWithConfigFlagName); f != nil && f.Value.String() == "true" {
		paths, err = s.applyConfig(dbConfig, "clickhouse", paths)
		if err != nil {
			return err
		}
		// remove tmp directory after migration
		defer func() {
			for _, srcPath := range paths {
				_ = os.RemoveAll(srcPath)
			}
		}()
	}

	// we count on that migrations are located either in "src" folder or in "src/click" folder
	return s.executeMigrationCmd(openDb, migration.DialectClickHouse, paths, up)
}

func (s *ServiceInstance[TCfg]) applyConfig(cfg any, prefix string, srcPaths []string) ([]string, error) {
	flatCfg := make(map[string]string)
	flattenStructToMapString(reflect.ValueOf(cfg), prefix, flatCfg)

	var resultPaths []string
	for _, srcPath := range srcPaths {
		// check folder exists
		absPath, _ := filepath.Abs(srcPath)
		if _, err := os.Stat(absPath); err == nil {

			files, err := os.ReadDir(srcPath)
			if err != nil {
				return nil, ErrApplyConfigReadDirFailed(err)
			}

			tmpDir, err := os.MkdirTemp(srcPath, tmpMigrationDirPrefix)
			if err != nil {
				return nil, ErrApplyConfigMkdirTempFailed(err)
			}

			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
					continue
				}

				origPath := filepath.Join(srcPath, f.Name())
				contentBytes, err := os.ReadFile(origPath)
				if err != nil {
					return nil, ErrApplyConfigReadFileFailed(err)
				}
				content := string(contentBytes)

				for k, v := range flatCfg {
					content = strings.ReplaceAll(content, "{{"+strings.ToLower(k)+"}}", v)
				}

				// Step 5: save the file
				newPath := filepath.Join(tmpDir, f.Name())
				if err := os.WriteFile(newPath, []byte(content), 0644); err != nil {
					return nil, ErrApplyConfigWriteFileFailed(err)
				}
			}

			resultPaths = append(resultPaths, tmpDir)
		}
	}

	return resultPaths, nil
}

func flattenStructToMapString(v reflect.Value, prefix string, out map[string]string) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() {
		return
	}

	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			fieldVal := v.Field(i)
			if !fieldVal.CanInterface() {
				continue
			}

			key := field.Name
			if tag, ok := field.Tag.Lookup("mapstructure"); ok {
				key = tag
			}
			key = strings.ToLower(key)

			fullKey := key
			if prefix != "" {
				fullKey = prefix + "_" + key
			}

			flattenStructToMapString(fieldVal, fullKey, out)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			mapKey := fmt.Sprintf("%v", k.Interface())
			mapVal := v.MapIndex(k)
			newPrefix := prefix + "_" + mapKey
			flattenStructToMapString(mapVal, newPrefix, out)
		}
	default:
		out[prefix] = fmt.Sprintf("%v", v.Interface())
	}
}

func (s *ServiceInstance[TCfg]) executeMigrationCmd(openDbFn func() (*sql.DB, error), dialect string, srcPaths []string, up bool) error {

	// check paths
	var srcPath string
	for _, p := range srcPaths {
		// check folder exists
		absPath, _ := filepath.Abs(p)
		if _, err := os.Stat(absPath); err == nil {
			srcPath = absPath
			break
		}
	}

	// open db
	sqlDb, err := openDbFn()
	if err != nil {
		return err
	}
	defer func() { _ = sqlDb.Close() }()

	// migration
	m := migration.NewMigration(sqlDb, srcPath, s.GetLogger(), dialect)

	// run migration command
	if up {
		return m.Up()
	}
	return m.Down()
}
