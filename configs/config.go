package configs

import (
    "os"
)

type ArangoConfig struct {
    Host     string
    Username string
    Password string
    Database string
}

func LoadArangoConfig() ArangoConfig {
    return ArangoConfig{
        Host:     getEnv("ARANGO_HOST", "http://localhost:8529"),
        Username: getEnv("ARANGO_USER", "root"),
        Password: getEnv("ARANGO_PASSWORD", "root"),
        Database: getEnv("ARANGO_DB", "userdb"),
    }
}

func getEnv(key, defaultVal string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultVal
}
