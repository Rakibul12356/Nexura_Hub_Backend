package config

import "nexura-backend/internal/core/config"

type Config = config.Config

func LoadConfig() *Config {
	return config.LoadConfig()
}
