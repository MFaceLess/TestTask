package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	RefillInterval        int     `json:"refill_interval"`
	ListenPort            int     `json:"listen_port"`
	BucketDefaultCapacity float64 `json:"bucket_default_capacity"`
	DefaultRefillRate     float64 `json:"default_refill_rate"`
}

func LoadConfig(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg Config

	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
