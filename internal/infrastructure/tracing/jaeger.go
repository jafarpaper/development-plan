package tracing

import (
	"fmt"
	"io"
	"time"

	"activity-log-service/internal/infrastructure/config"
	"github.com/opentracing/opentracing-go"
	jaegerConfig "github.com/uber/jaeger-client-go/config"
	jaegerLog "github.com/uber/jaeger-client-go/log"
	"github.com/uber/jaeger-lib/metrics"
)

func InitJaeger(cfg *config.JaegerConfig) (opentracing.Tracer, io.Closer, error) {
	jaegerCfg := jaegerConfig.Configuration{
		ServiceName: cfg.ServiceName,
		Sampler: &jaegerConfig.SamplerConfig{
			Type:  cfg.SamplerType,
			Param: cfg.SamplerParam,
		},
		Reporter: &jaegerConfig.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  cfg.Endpoint,
		},
	}

	jLogger := jaegerLog.StdLogger
	jMetricsFactory := metrics.NullFactory

	tracer, closer, err := jaegerCfg.NewTracer(
		jaegerConfig.Logger(jLogger),
		jaegerConfig.Metrics(jMetricsFactory),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize jaeger tracer: %w", err)
	}

	opentracing.SetGlobalTracer(tracer)
	return tracer, closer, nil
}
