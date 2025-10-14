package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type (
	Gateway struct {
		// Prefix stores prefix to send request in target
		Prefix string `yaml:"prefix"`
		// Target stores url of proxy target
		Target string `yaml:"target"`
		// Auth stores auth off or on
		Auth bool `yaml:"auth"`
		// Cash stores cashing off or on
		Cash bool `yaml:"cash"`
	}

	AuthConfig struct {
		// Key stores key for jwt
		Key string `yaml:"key"`
	}

	RateLimitingConfig struct {
		MaxRequest int `yaml:"max_req"`
	}

	CORSConfig struct {
		AllowMethods []string `yaml:"allow_methods"`
		AllowHeaders []string `yaml:"allow_headers"`
		MaxAge       int      `yaml:"max_age"`
	}

	Config struct {
		// Addr stores address of api gateway
		Addr string `yaml:"addr"`
		// CORS off or on
		CORS bool `yaml:"cors"`
		// CORSConfig stores config for CORS
		CorsConfig CORSConfig `yaml:"cors_config"`
		// Gateway stores all gateways to route
		Gateways []Gateway `yaml:"gateways"`
	}
)

func NewConfig(path string) (*Config, error) {
	config := Config{}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed open config file %s: %v", path, err)
	}

	if err = yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("faield decode config: %v", err)
	}

	return &config, nil
}
