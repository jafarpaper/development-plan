package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Arango  ArangoConfig  `mapstructure:"arango"`
	NATS    NATSConfig    `mapstructure:"nats"`
	Logger  LoggerConfig  `mapstructure:"logger"`
	Jaeger  JaegerConfig  `mapstructure:"jaeger"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Redis   RedisConfig   `mapstructure:"redis"`
	Email   EmailConfig   `mapstructure:"email"`
	Cron    CronConfig    `mapstructure:"cron"`
}

type ServerConfig struct {
	Port              int           `mapstructure:"port"`
	GRPCPort          int           `mapstructure:"grpc_port"`
	Timeout           time.Duration `mapstructure:"timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	MaxConnectionIdle time.Duration `mapstructure:"max_connection_idle"`
	MaxConnectionAge  time.Duration `mapstructure:"max_connection_age"`
}

type ArangoConfig struct {
	URL        string `mapstructure:"url"`
	Database   string `mapstructure:"database"`
	Username   string `mapstructure:"username"`
	Password   string `mapstructure:"password"`
	Collection string `mapstructure:"collection"`
}

type NATSConfig struct {
	URL            string        `mapstructure:"url"`
	Stream         string        `mapstructure:"stream"`
	Subject        string        `mapstructure:"subject"`
	Durable        string        `mapstructure:"durable"`
	DeliverSubject string        `mapstructure:"deliver_subject"`
	AckWait        time.Duration `mapstructure:"ack_wait"`
	MaxDeliver     int           `mapstructure:"max_deliver"`
}

type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

type JaegerConfig struct {
	ServiceName  string  `mapstructure:"service_name"`
	Endpoint     string  `mapstructure:"endpoint"`
	SamplerType  string  `mapstructure:"sampler_type"`
	SamplerParam float64 `mapstructure:"sampler_param"`
}

type MetricsConfig struct {
	Port int    `mapstructure:"port"`
	Path string `mapstructure:"path"`
}

type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type EmailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	Enabled  bool   `mapstructure:"enabled"`
}

type CronConfig struct {
	DailySummaryTime string `mapstructure:"daily_summary_time"`
	CleanupInterval  string `mapstructure:"cleanup_interval"`
	Enabled          bool   `mapstructure:"enabled"`
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)

	viper.AutomaticEnv()

	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.grpc_port", 9000)
	viper.SetDefault("server.timeout", "30s")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.max_connection_idle", "5m")
	viper.SetDefault("server.max_connection_age", "5m")

	viper.SetDefault("arango.url", "http://localhost:8529")
	viper.SetDefault("arango.database", "activity_logs")
	viper.SetDefault("arango.username", "root")
	viper.SetDefault("arango.password", "rootpassword")
	viper.SetDefault("arango.collection", "activity_log")

	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.stream", "ACTIVITY_LOGS")
	viper.SetDefault("nats.subject", "activity.log.created")
	viper.SetDefault("nats.durable", "activity-log-consumer")
	viper.SetDefault("nats.deliver_subject", "activity.log.deliver")
	viper.SetDefault("nats.ack_wait", "30s")
	viper.SetDefault("nats.max_deliver", 3)

	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "json")
	viper.SetDefault("logger.output", "stdout")

	viper.SetDefault("jaeger.service_name", "activity-log-service")
	viper.SetDefault("jaeger.endpoint", "http://localhost:14268/api/traces")
	viper.SetDefault("jaeger.sampler_type", "const")
	viper.SetDefault("jaeger.sampler_param", 1.0)

	viper.SetDefault("metrics.port", 2112)
	viper.SetDefault("metrics.path", "/metrics")

	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	viper.SetDefault("email.host", "localhost")
	viper.SetDefault("email.port", 1025)
	viper.SetDefault("email.username", "")
	viper.SetDefault("email.password", "")
	viper.SetDefault("email.from", "activity-log-service@example.com")
	viper.SetDefault("email.enabled", true)

	viper.SetDefault("cron.daily_summary_time", "08:00")
	viper.SetDefault("cron.cleanup_interval", "24h")
	viper.SetDefault("cron.enabled", true)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
