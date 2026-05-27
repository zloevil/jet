package cluster

import (
	_ "embed"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
	"github.com/zloevil/jet"
	"github.com/zloevil/jet/storages/clickhouse"
	"os"
	"reflect"
	"testing"
)

type TestConfigType struct{}

type serviceTestSuite struct {
	jet.Suite
	serviceInstance *ServiceInstance[TestConfigType]
	cmd             *cobra.Command
}

func (s *serviceTestSuite) SetupSuite() {
	s.Suite.Init(func() jet.CLogger { return jet.L(jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel})) })
}

func (s *serviceTestSuite) SetupTest() {
	s.serviceInstance = &ServiceInstance[TestConfigType]{
		migSourceEnv: func() *string { v := "TEST_MIGRATION_SOURCE"; return &v }(),
		logger:       jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel, Format: jet.FormatterJson}),
	}
	s.cmd = &cobra.Command{}
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(serviceTestSuite))
}

func (s *serviceTestSuite) Test_LoadMigrationSources_WhenFlagProvided_SingleSource() {

	s.cmd.Flags().StringArrayVar(&[]string{}, migrationSourceFlagName, []string{"schema1:/usr"}, "")

	res, err := s.serviceInstance.loadMigrationSources(s.cmd)
	s.NoError(err)

	s.Equal(1, len(res))
	s.Equal("/usr", res["schema1"])
}

func (s *serviceTestSuite) Test_LoadMigrationSources_WhenFlagProvided_SingleSourceWithoutSchema() {

	s.cmd.Flags().StringArrayVar(&[]string{}, migrationSourceFlagName, []string{"/usr"}, "")

	res, err := s.serviceInstance.loadMigrationSources(s.cmd)
	s.NoError(err)

	s.Equal(1, len(res))
	s.Equal("/usr", res["default"])
}

func (s *serviceTestSuite) Test_LoadMigrationSources_WhenFlagProvided_MultipleSources() {

	sources := []string{"schema1:/usr", "schema2:/var"}
	s.cmd.Flags().StringArrayVar(&[]string{}, migrationSourceFlagName, sources, "")

	res, err := s.serviceInstance.loadMigrationSources(s.cmd)
	s.NoError(err)

	s.Equal(2, len(res))
	s.Equal("/usr", res["schema1"])
	s.Equal("/var", res["schema2"])
}

func (s *serviceTestSuite) Test_LoadMigrationSources_WhenFlagProvided_InvalidPath() {

	sources := []string{"schema1:/usr", "schema2:/someunknownpath"}
	s.cmd.Flags().StringArrayVar(&[]string{}, migrationSourceFlagName, sources, "")

	_, err := s.serviceInstance.loadMigrationSources(s.cmd)
	s.AssertAppErr(err, ErrCodeMigrationSourcePathInvalid)
}

func (s *serviceTestSuite) Test_LoadMigrationSources_WhenFlagEmpty_EnvVariableProvided_SingleSource() {

	s.cmd.Flags().StringArrayVar(&[]string{}, migrationSourceFlagName, []string{}, "")

	_ = os.Setenv("TEST_MIGRATION_SOURCE", "schema1:/usr")
	defer os.Unsetenv("TEST_MIGRATION_SOURCE")

	res, err := s.serviceInstance.loadMigrationSources(s.cmd)
	s.NoError(err)

	s.Equal(1, len(res))
	s.Equal("/usr", res["schema1"])
}

func (s *serviceTestSuite) Test_LoadMigrationSources_WhenFlagEmpty_EnvVariableProvided_MultipleSources() {

	s.cmd.Flags().StringArrayVar(&[]string{}, migrationSourceFlagName, []string{}, "")

	_ = os.Setenv("TEST_MIGRATION_SOURCE", "schema1:/usr,schema2:/var")
	defer os.Unsetenv("TEST_MIGRATION_SOURCE")

	res, err := s.serviceInstance.loadMigrationSources(s.cmd)
	s.NoError(err)

	s.Equal(2, len(res))
	s.Equal("/usr", res["schema1"])
	s.Equal("/var", res["schema2"])
}

func (s *serviceTestSuite) Test_FlattenStructToMapString() {
	flatCfg := make(map[string]string)

	cfg := clickhouse.Config{
		Engines: &clickhouse.Engines{Kafka: map[string]*clickhouse.KafkaEngine{
			"resources": {
				BrokerList:   "broker",
				TopicList:    "topic.1.1.2",
				GroupName:    "test.group",
				NumConsumers: 1,
			},
		}},
	}

	flattenStructToMapString(reflect.ValueOf(cfg), "clickhouse", flatCfg)

	s.Equal("broker", flatCfg["clickhouse_engines_kafka_resources_broker_list"])
	s.Equal("topic.1.1.2", flatCfg["clickhouse_engines_kafka_resources_topic_list"])
	s.Equal("test.group", flatCfg["clickhouse_engines_kafka_resources_group_name"])
	s.Equal("1", flatCfg["clickhouse_engines_kafka_resources_num_consumers"])
}

var (
	//go:embed expected.sql
	expected []byte
)

func (s *serviceTestSuite) Test_ApplyEnginesToScript() {
	serviceInstance := &ServiceInstance[clickhouse.Config]{
		migSourceEnv: func() *string { v := "TEST_MIGRATION_SOURCE"; return &v }(),
		logger:       jet.InitLogger(&jet.LogConfig{Level: jet.TraceLevel, Format: jet.FormatterJson}),
	}

	res, err := serviceInstance.applyConfig(clickhouse.Config{
		Engines: &clickhouse.Engines{Kafka: map[string]*clickhouse.KafkaEngine{
			"resources": {
				BrokerList:   "broker",
				TopicList:    "topic.1.1.2",
				GroupName:    "test.group",
				NumConsumers: 1,
			},
		}},
	}, "clickhouse", []string{"./migrations"})
	s.NoError(err)

	defer func() {
		for _, srcPath := range res {
			_ = os.RemoveAll(srcPath)
		}
	}()

	s.Len(res, 1)
	s.Contains(res[0], "migrations/"+tmpMigrationDirPrefix)

	content, err := os.ReadFile(res[0] + "/test.sql")
	s.NoError(err)
	s.Equal(expected, content)
}

func (s *serviceTestSuite) Test_LoadMigrationSources_WhenFlagEmpty_EnvVariableProvided_InvalidPath() {

	s.cmd.Flags().StringArrayVar(&[]string{}, migrationSourceFlagName, []string{}, "")

	_ = os.Setenv("TEST_MIGRATION_SOURCE", "schema1:/usr,schema2:/someunknownpath")
	defer os.Unsetenv("TEST_MIGRATION_SOURCE")

	_, err := s.serviceInstance.loadMigrationSources(s.cmd)
	s.AssertAppErr(err, ErrCodeMigrationSourcePathInvalid)
}

func (s *serviceTestSuite) Test_LoadMigrationSources_WhenFlagAndEnvBothProvided_TakeFromFlag() {

	sources := []string{"schema1:/usr"}
	s.cmd.Flags().StringArrayVar(&[]string{}, migrationSourceFlagName, sources, "")

	os.Setenv("TEST_MIGRATION_SOURCE", "schema2:/var")
	defer os.Unsetenv("TEST_MIGRATION_SOURCE")

	res, err := s.serviceInstance.loadMigrationSources(s.cmd)
	s.NoError(err)

	s.Equal(1, len(res))
	s.Equal("/usr", res["schema1"])
}

func (s *serviceTestSuite) Test_LoadMigrationSources_WhenInvalidFlagFormat() {

	sources := []string{"schema1:path:extra"}
	s.cmd.Flags().StringArrayVar(&[]string{}, migrationSourceFlagName, sources, "")

	_, err := s.serviceInstance.loadMigrationSources(s.cmd)
	s.AssertAppErr(err, ErrCodeMigrationSourceParamInvalid)

}

func (s *serviceTestSuite) Test_LoadMigrationSources_WhenNoFlagOrEnvProvided() {
	s.cmd.Flags().StringArrayVar(&[]string{}, migrationSourceFlagName, []string{}, "")
	os.Setenv("TEST_MIGRATION_SOURCE", "")
	defer os.Unsetenv("TEST_MIGRATION_SOURCE")
	_, err := s.serviceInstance.loadMigrationSources(s.cmd)
	s.NoError(err)
}
