package initialization

import (
	"context"
	"fmt"

	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"

	"activity-log-service/internal/application/usecase"
	"activity-log-service/internal/domain/repository"
	"activity-log-service/internal/infrastructure/cache"
	"activity-log-service/internal/infrastructure/config"
	"activity-log-service/internal/infrastructure/database"
	"activity-log-service/internal/infrastructure/email"
	"activity-log-service/internal/infrastructure/messaging"
	infraRepo "activity-log-service/internal/infrastructure/repository"
	"activity-log-service/internal/infrastructure/tracing"
)

// Dependencies holds all initialized dependencies
type Dependencies struct {
	Config       *config.Config
	Logger       *logrus.Logger
	Tracer       opentracing.Tracer
	TracerCloser func() error
	Repository   repository.ActivityLogRepository
	Cache        *cache.RedisCache
	Publisher    *messaging.NATSPublisher
	Mailer       *email.Mailer
	UseCase      *usecase.ActivityLogUseCase
}

// InitializationOptions holds optional configurations for initialization
type InitializationOptions struct {
	ConfigPath        string
	RequireCache      bool
	RequireEmail      bool
	RequireNATS       bool
	MetricsPortOffset int
}

// Initialize sets up all application dependencies
func Initialize(opts *InitializationOptions) (*Dependencies, error) {
	if opts == nil {
		opts = &InitializationOptions{}
	}

	deps := &Dependencies{}

	// Load configuration
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	deps.Config = cfg

	// Setup logger
	logger := logrus.New()
	logger.SetLevel(getLogLevel(cfg.Logger.Level))
	if cfg.Logger.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	}
	deps.Logger = logger

	// Initialize tracing
	tracer, closer, err := tracing.InitJaeger(&cfg.Jaeger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Jaeger tracer: %w", err)
	}
	deps.Tracer = tracer
	deps.TracerCloser = closer.Close

	// Initialize ArangoDB repository
	arangoRepo, err := database.NewArangoActivityLogRepository(
		cfg.Arango.URL,
		cfg.Arango.Database,
		cfg.Arango.Collection,
		cfg.Arango.Username,
		cfg.Arango.Password,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ArangoDB repository: %w", err)
	}

	// Initialize Redis cache (optional)
	var finalRepo repository.ActivityLogRepository = arangoRepo
	if cfg.Redis.Address != "" {
		redisCache := cache.NewRedisCache(cache.CacheConfig{
			Address:  cfg.Redis.Address,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}, logger)

		if err := redisCache.Ping(context.Background()); err != nil {
			if opts.RequireCache {
				return nil, fmt.Errorf("failed to connect to Redis cache: %w", err)
			}
			logger.WithError(err).Warn("Failed to connect to Redis cache, using direct repository")
		} else {
			finalRepo = infraRepo.NewCachedActivityLogRepository(arangoRepo, redisCache, logger)
			deps.Cache = redisCache
			logger.Info("Redis cache enabled")
		}
	} else if opts.RequireCache {
		return nil, fmt.Errorf("Redis configuration is required but not provided")
	}
	deps.Repository = finalRepo

	// Initialize NATS publisher (optional)
	if cfg.NATS.URL != "" || opts.RequireNATS {
		if cfg.NATS.URL == "" {
			return nil, fmt.Errorf("NATS configuration is required but not provided")
		}

		publisher, err := messaging.NewNATSPublisher(cfg.NATS.URL, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create NATS publisher: %w", err)
		}

		// Ensure NATS stream exists
		if err := publisher.EnsureStream(cfg.NATS.Stream, cfg.NATS.Subject); err != nil {
			return nil, fmt.Errorf("failed to ensure NATS stream: %w", err)
		}

		deps.Publisher = publisher
	}

	// Initialize email service (optional)
	if cfg.Email.Enabled || opts.RequireEmail {
		if !cfg.Email.Enabled {
			return nil, fmt.Errorf("email service is required but not enabled in config")
		}

		mailer := email.NewMailer(email.EmailConfig{
			Host:     cfg.Email.Host,
			Port:     cfg.Email.Port,
			Username: cfg.Email.Username,
			Password: cfg.Email.Password,
			From:     cfg.Email.From,
		}, logger)
		deps.Mailer = mailer
		logger.Info("Email service enabled")
	}

	// Initialize use case
	deps.UseCase = usecase.NewActivityLogUseCase(finalRepo, deps.Publisher, deps.Mailer)

	return deps, nil
}

// Cleanup properly closes all connections and resources
func (d *Dependencies) Cleanup() error {
	var errors []error

	if d.Publisher != nil {
		if err := d.Publisher.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close NATS publisher: %w", err))
		}
	}

	if d.Cache != nil {
		if err := d.Cache.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close Redis cache: %w", err))
		}
	}

	if d.TracerCloser != nil {
		if err := d.TracerCloser(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close tracer: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// GetHTTPDependencies returns dependencies needed for HTTP server
func GetHTTPDependencies(configPath string) (*Dependencies, error) {
	return Initialize(&InitializationOptions{
		ConfigPath:        configPath,
		RequireNATS:       true,
		RequireEmail:      false,
		RequireCache:      false,
		MetricsPortOffset: 1,
	})
}

// GetGRPCDependencies returns dependencies needed for gRPC server
func GetGRPCDependencies(configPath string) (*Dependencies, error) {
	return Initialize(&InitializationOptions{
		ConfigPath:        configPath,
		RequireNATS:       false,
		RequireEmail:      false,
		RequireCache:      false,
		MetricsPortOffset: 0,
	})
}

// GetConsumerDependencies returns dependencies needed for NATS consumer
func GetConsumerDependencies(configPath string) (*Dependencies, error) {
	return Initialize(&InitializationOptions{
		ConfigPath:        configPath,
		RequireNATS:       false,
		RequireEmail:      false,
		RequireCache:      false,
		MetricsPortOffset: 2,
	})
}

// GetCronDependencies returns dependencies needed for cron server
func GetCronDependencies(configPath string) (*Dependencies, error) {
	return Initialize(&InitializationOptions{
		ConfigPath:        configPath,
		RequireNATS:       false,
		RequireEmail:      false,
		RequireCache:      true,
		MetricsPortOffset: 3,
	})
}

func getLogLevel(level string) logrus.Level {
	switch level {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}
