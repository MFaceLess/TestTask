package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	ListenPort          int      `json:"listen_port"`
	Algorithm           string   `json:"algorithm"`
	Backends            []string `json:"backends"`
	HealthCheckInterval int      `json:"health_check_interval"`
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
