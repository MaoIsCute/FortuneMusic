package config

import (
    "log"
    "os"
    "github.com/joho/godotenv"
)

type Config struct {
    DBUrl              string
    GoogleClientID     string
    GoogleClientSecret string
    JWTSecret          string
    AppURL             string
}

func Load() *Config {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
    return &Config{
        DBUrl:              os.Getenv("DATABASE_URL"),
        GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
        GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
        JWTSecret:          os.Getenv("JWT_SECRET"),
        AppURL:             os.Getenv("APP_URL"),
    }
}
