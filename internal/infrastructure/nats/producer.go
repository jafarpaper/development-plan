package nats

import (
    "github.com/nats-io/nats.go"
    "log"
)

type Producer struct {
    nc *nats.Conn
}

func NewProducer() (*Producer, error) {
    nc, err := nats.Connect(nats.DefaultURL)
    if err != nil {
        return nil, err
    }
    return &Producer{nc: nc}, nil
}

func (p *Producer) PublishUserEvent(userID string) {
    err := p.nc.Publish("user.events", []byte("User fetched: "+userID))
    if err != nil {
        log.Printf("Failed to publish event: %v", err)
    }
}
