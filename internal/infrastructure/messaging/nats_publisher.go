package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"

	"activity-log-service/internal/domain/event"
)

type NATSPublisher struct {
	conn   *nats.Conn
	js     nats.JetStreamContext
	logger *logrus.Logger
}

func NewNATSPublisher(url string, logger *logrus.Logger) (*NATSPublisher, error) {
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

	return &NATSPublisher{
		conn:   conn,
		js:     js,
		logger: logger,
	}, nil
}

func (p *NATSPublisher) PublishActivityLogCreated(ctx context.Context, event *event.ActivityLogCreated) error {
	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	msg := &nats.Msg{
		Subject: "activity.log.created",
		Data:    data,
		Header:  make(nats.Header),
	}

	msg.Header.Set("event-type", event.GetEventType())
	msg.Header.Set("aggregate-id", event.GetAggregateID())
	msg.Header.Set("timestamp", event.GetTimestamp().Format(time.RFC3339))

	_, err = p.js.PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"event_type":   event.GetEventType(),
		"aggregate_id": event.GetAggregateID(),
		"subject":      msg.Subject,
	}).Info("Event published successfully")

	return nil
}

func (p *NATSPublisher) Close() error {
	p.conn.Close()
	return nil
}

func (p *NATSPublisher) EnsureStream(streamName, subject string) error {
	stream, err := p.js.StreamInfo(streamName)
	if err != nil {
		if err == nats.ErrStreamNotFound {
			_, err = p.js.AddStream(&nats.StreamConfig{
				Name:      streamName,
				Subjects:  []string{subject},
				Retention: nats.LimitsPolicy,
				MaxAge:    time.Hour * 24 * 30,
				MaxMsgs:   1000000,
				Storage:   nats.FileStorage,
			})
			if err != nil {
				return fmt.Errorf("failed to create stream: %w", err)
			}
			p.logger.WithField("stream", streamName).Info("Stream created")
		} else {
			return fmt.Errorf("failed to get stream info: %w", err)
		}
	} else {
		p.logger.WithField("stream", stream.Config.Name).Info("Stream already exists")
	}

	return nil
}
