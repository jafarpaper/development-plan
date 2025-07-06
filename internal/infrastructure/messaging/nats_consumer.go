package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/sirupsen/logrus"

	"activity-log-service/internal/domain/event"
	"activity-log-service/internal/domain/repository"
)

type NATSConsumer struct {
	conn         *nats.Conn
	js           nats.JetStreamContext
	logger       *logrus.Logger
	arangoRepo   repository.ActivityLogRepository
	subscription *nats.Subscription
	workerPool   *WorkerPool
	stopCh       chan struct{}
	wg           sync.WaitGroup
	tracer       opentracing.Tracer
}

type ActivityLogHandler func(ctx context.Context, event *event.ActivityLogCreated) error

func NewNATSConsumer(
	url string,
	logger *logrus.Logger,
	arangoRepo repository.ActivityLogRepository,
	workers int,
	tracer opentracing.Tracer,
) (*NATSConsumer, error) {
	conn, err := nats.Connect(url,
		nats.ReconnectWait(time.Second*2),
		nats.MaxReconnects(10),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.WithError(err).Error("NATS disconnected")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	workerPool := NewWorkerPool(workers, logger)

	return &NATSConsumer{
		conn:       conn,
		js:         js,
		logger:     logger,
		arangoRepo: arangoRepo,
		workerPool: workerPool,
		stopCh:     make(chan struct{}),
		tracer:     tracer,
	}, nil
}

func (c *NATSConsumer) Start(ctx context.Context) error {
	c.workerPool.Start()

	sub, err := c.js.Subscribe("activity.log.created", c.handleMessage, nats.Durable("activity-log-consumer"))
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	c.subscription = sub

	c.logger.Info("NATS consumer started")

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		<-ctx.Done()
		c.Stop()
	}()

	return nil
}

func (c *NATSConsumer) Stop() {
	c.logger.Info("Stopping NATS consumer")

	if c.subscription != nil {
		c.subscription.Unsubscribe()
	}

	c.workerPool.Stop()
	close(c.stopCh)
	c.conn.Close()

	c.logger.Info("NATS consumer stopped")
}

func (c *NATSConsumer) handleMessage(msg *nats.Msg) {
	job := &Job{
		ID:   fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Data: msg.Data,
		Handler: func(ctx context.Context, data []byte) error {
			return c.processActivityLogEvent(ctx, data)
		},
		OnSuccess: func() {
			msg.Ack()
			c.logger.Debug("Message acknowledged")
		},
		OnError: func(err error) {
			c.logger.WithError(err).Error("Failed to process message")
			msg.Nak()
		},
	}

	c.workerPool.Submit(job)
}

func (c *NATSConsumer) processActivityLogEvent(ctx context.Context, data []byte) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "processActivityLogEvent")
	defer span.Finish()

	ext.Component.Set(span, "nats-consumer")

	var event event.ActivityLogCreated
	if err := json.Unmarshal(data, &event); err != nil {
		ext.Error.Set(span, true)
		span.SetTag("error.message", err.Error())
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	span.SetTag("event_type", event.GetEventType())
	span.SetTag("aggregate_id", event.GetAggregateID())

	c.logger.WithFields(logrus.Fields{
		"event_type":   event.GetEventType(),
		"aggregate_id": event.GetAggregateID(),
	}).Info("Processing activity log event")

	if err := c.arangoRepo.Create(ctx, event.ActivityLog); err != nil {
		ext.Error.Set(span, true)
		span.SetTag("error.message", err.Error())
		return fmt.Errorf("failed to save to ArangoDB: %w", err)
	}

	return nil
}

func (c *NATSConsumer) Wait() {
	c.wg.Wait()
}

type WorkerPool struct {
	workers  int
	jobQueue chan *Job
	quit     chan struct{}
	logger   *logrus.Logger
	wg       sync.WaitGroup
}

type Job struct {
	ID        string
	Data      []byte
	Handler   func(ctx context.Context, data []byte) error
	OnSuccess func()
	OnError   func(error)
}

func NewWorkerPool(workers int, logger *logrus.Logger) *WorkerPool {
	return &WorkerPool{
		workers:  workers,
		jobQueue: make(chan *Job, 100),
		quit:     make(chan struct{}),
		logger:   logger,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	wp.logger.WithField("workers", wp.workers).Info("Worker pool started")
}

func (wp *WorkerPool) Stop() {
	close(wp.quit)
	wp.wg.Wait()
	wp.logger.Info("Worker pool stopped")
}

func (wp *WorkerPool) Submit(job *Job) {
	select {
	case wp.jobQueue <- job:
	case <-wp.quit:
		wp.logger.Warn("Worker pool is shutting down, job rejected")
	}
}

func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	logger := wp.logger.WithField("worker_id", id)
	logger.Info("Worker started")

	for {
		select {
		case job := <-wp.jobQueue:
			logger.WithField("job_id", job.ID).Debug("Processing job")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := job.Handler(ctx, job.Data)
			cancel()

			if err != nil {
				logger.WithError(err).WithField("job_id", job.ID).Error("Job failed")
				if job.OnError != nil {
					job.OnError(err)
				}
			} else {
				logger.WithField("job_id", job.ID).Debug("Job completed successfully")
				if job.OnSuccess != nil {
					job.OnSuccess()
				}
			}

		case <-wp.quit:
			logger.Info("Worker stopping")
			return
		}
	}
}
