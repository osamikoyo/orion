package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type (
	// Target stores target info
	Target struct {
		// Url stores target url
		Url string
		// HealthEndpoint stores endpoint for healthchecker
		// Body of get request must be "OK" or "NOT"
		HealthEndpoint string `yaml:"health_endpoint"`
	}

	TLS struct {
		Cert string `yaml:"cert"`
		Key  string `yaml:"key"`
	}

	Gateway struct {
		// Prefix stores prefix to send request in target
		Prefix string `yaml:"prefix"`
		// Target stores url of proxy target
		Targets []Target `yaml:"targets"`
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
		Use        bool `yaml:"use"`
		MaxRequest int  `yaml:"max_req"`
	}

	CORSConfig struct {
		AllowMethods []string `yaml:"allow_methods"`
		AllowHeaders []string `yaml:"allow_headers"`
		MaxAge       int      `yaml:"max_age"`
	}

	Config struct {
		// Addr stores address of api gateway
		Addr string `yaml:"addr"`
		// Proto stores protocol, it can be http or http3
		Proto string `yaml:"proto"`

		Tls TLS `yaml:"tlc"`
		// AuthConfig stores config for auth
		AuthConfig AuthConfig `yaml:"auth"`
		// CORS off or on
		CORS bool `yaml:"cors"`
		// CORSConfig stores config for CORS
		CorsConfig CORSConfig `yaml:"cors_config"`
		// RLconfig stores config for rate limiting
		RateLimitingConfig RateLimitingConfig `yaml:"rate_limiting"`
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
