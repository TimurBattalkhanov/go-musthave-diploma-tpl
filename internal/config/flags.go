package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"os"
)

type AppConfig struct {
	Address        string
	DatabaseURI    string
	AccrualAddress string
	Secret         string
}

func NewAppConfig() AppConfig {
	var cfg AppConfig
	flag.StringVar(&cfg.Address, "a", "localhost:8080", "The address to listen on")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "The database url")
	flag.StringVar(&cfg.AccrualAddress, "r", "", "The accrual address")
	flag.StringVar(&cfg.Secret, "s", "", "The secret")
	flag.Parse()

	if v := os.Getenv("RUN_ADDRESS"); v != "" {
		cfg.Address = v
	}

	if v := os.Getenv("DATABASE_URI"); v != "" {
		cfg.DatabaseURI = v
	}

	if v := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); v != "" {
		cfg.AccrualAddress = v
	}

	if v := os.Getenv("SECRET"); v != "" {
		cfg.Secret = v
	}
	if cfg.Secret == "" {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			panic(err)
		}
		cfg.Secret = hex.EncodeToString(b)
	}

	return cfg
}
