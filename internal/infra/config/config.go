package config

import (
	"sync"

	"github.com/spf13/viper"
)

var (
	once    sync.Once
	global  *Config
	loadErr error
)

type Config struct {
	ServerPort      int
	PprofPort       int
	DBDSN           string
	DBAdminDSN      string
	OTelEnabled     bool
	OTelEndpoint    string
	OTelServiceName    string
	OTelServiceVersion string
	OTelEnvironment    string
	OTelSampleRate     float64
	GeneralLogLevel string
}

func Get() (*Config, error) {
	once.Do(load)
	return global, loadErr
}

func load() {
	v := viper.New()
	v.SetConfigName("server")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/audit-log")
	_ = v.ReadInConfig()

	v.SetEnvPrefix("AUDIT_LOG")
	v.AutomaticEnv()

	v.SetDefault("server_port", 50051)
	v.SetDefault("server_pprof_port", 6061)
	v.SetDefault("otel_enabled", true)
	v.SetDefault("otel_endpoint", "localhost:4317")
	v.SetDefault("otel_service_name", "audit-log")
	v.SetDefault("otel_service_version", "")
	v.SetDefault("otel_environment", "development")
	v.SetDefault("otel_sample_rate", 0.1)
	v.SetDefault("general_log_level", "info")

	global = &Config{
		ServerPort:      v.GetInt("server_port"),
		PprofPort:       v.GetInt("server_pprof_port"),
		DBDSN:           v.GetString("db_dsn"),
		DBAdminDSN:      v.GetString("db_admin_dsn"),
		OTelEnabled:     v.GetBool("otel_enabled"),
		OTelEndpoint:    v.GetString("otel_endpoint"),
		OTelServiceName:    v.GetString("otel_service_name"),
		OTelServiceVersion: v.GetString("otel_service_version"),
		OTelEnvironment:    v.GetString("otel_environment"),
		OTelSampleRate:  v.GetFloat64("otel_sample_rate"),
		GeneralLogLevel: v.GetString("general_log_level"),
	}
}
