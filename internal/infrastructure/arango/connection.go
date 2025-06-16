package arango

import (
    "context"
    "fmt"

    driver "github.com/arangodb/go-driver"
    "github.com/arangodb/go-driver/http"
    "go-clean-arango/configs"
)

type ArangoClient struct {
    Client driver.Client
    DB     driver.Database
}

func NewArangoClient(cfg configs.ArangoConfig) (*ArangoClient, error) {
    conn, err := http.NewConnection(http.ConnectionConfig{
        Endpoints: []string{cfg.Host},
    })
    if err != nil {
        return nil, fmt.Errorf("create connection error: %w", err)
    }

    client, err := driver.NewClient(driver.ClientConfig{
        Connection:     conn,
        Authentication: driver.BasicAuthentication(cfg.Username, cfg.Password),
    })
    if err != nil {
        return nil, fmt.Errorf("create client error: %w", err)
    }

    db, err := client.Database(context.Background(), cfg.Database)
    if err != nil {
        return nil, fmt.Errorf("connect to DB error: %w", err)
    }

    return &ArangoClient{Client: client, DB: db}, nil
}
