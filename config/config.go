package config

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Default values
const (
	DefaultAddr               = ":8080"
	DefaultProto              = "http"
	DefaultRequestTimeout     = 30 * time.Second
	DefaultHealthCheckTimeout = 5 * time.Second
	DefaultLoadBalancer       = "wrr"
	DefaultRateLimitMaxReq    = 100
	DefaultCORSMaxAge         = 86400
)

type Target struct {
	Url            string `yaml:"url" validate:"required,url"`
	Weight         int    `yaml:"weight" validate:"min=1"`
	HealthEndpoint string `yaml:"health_endpoint" validate:"omitempty"`
}

type TLS struct {
	Cert string `yaml:"cert" validate:"omitempty,required_with=Key,file"`
	Key  string `yaml:"key" validate:"omitempty,required_with=Cert,file"`
}

type Gateway struct {
	Prefix  string   `yaml:"prefix" validate:"required,startswith=/"`
	Targets []Target `yaml:"targets" validate:"min=1,dive"`
	Auth    bool     `yaml:"auth"`
	Cache   bool     `yaml:"cache"`
	Rate    bool     `yaml:"rate"`
}

type WafConfig struct {
	Use        bool   `yaml:"use"`
	ConfigPath string `yaml:"config_path" validate:"omitempty,file"`
}

type AuthConfig struct {
	Key string `yaml:"key" validate:"required_if=Gateways.Auth true"`
}

type RateLimitingConfig struct {
	MaxRequest int `yaml:"max_request" validate:"min=1"`
}

type CORSConfig struct {
	Use          bool     `yaml:"use"`
	AllowOrigins []string `yaml:"allow_origins" validate:"omitempty,dive,hostname|startswith=*"`
	AllowMethods []string `yaml:"allow_methods" validate:"omitempty,dive,oneof=GET POST PUT DELETE PATCH OPTIONS HEAD"`
	AllowHeaders []string `yaml:"allow_headers"`
	MaxAge       int      `yaml:"max_age" validate:"min=0,max=86400"`
}

type Config struct {
	Addr               string             `yaml:"addr" env:"GATEWAY_ADDR"`
	Proto              string             `yaml:"proto" env:"GATEWAY_PROTO" validate:"oneof=http http3"`
	RequestTimeout     time.Duration      `yaml:"request_timeout" env:"GATEWAY_REQ_TIMEOUT" validate:"min=1s"`
	LoadBalancerAlg    string             `yaml:"balancer" env:"GATEWAY_BALANCER" validate:"oneof=roundrobin wrr leastconn iphash"`
	TLS                TLS                `yaml:"tls"`
	WAF                WafConfig          `yaml:"waf"`
	AuthConfig         AuthConfig         `yaml:"auth"`
	HealthCheckTimeout time.Duration      `yaml:"hc_timeout" env:"GATEWAY_HC_TIMEOUT" validate:"min=1s"`
	CORS               CORSConfig         `yaml:"cors"`
	RateLimiting       RateLimitingConfig `yaml:"rate_limiting"`
	Gateways           []Gateway          `yaml:"gateways"`

	filePath string
}

func NewConfig(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", path, err)
	}

	cfg := &Config{filePath: path}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %v", err)
	}

	cfg.applyDefaults()

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env variables: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %v", err)
	}

	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Addr == "" {
		c.Addr = DefaultAddr
	}
	if c.Proto == "" {
		c.Proto = DefaultProto
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = DefaultRequestTimeout
	}
	if c.HealthCheckTimeout == 0 {
		c.HealthCheckTimeout = DefaultHealthCheckTimeout
	}
	if c.LoadBalancerAlg == "" {
		c.LoadBalancerAlg = DefaultLoadBalancer
	}
	if c.RateLimiting.MaxRequest == 0 {
		c.RateLimiting.MaxRequest = DefaultRateLimitMaxReq
	}
	if c.CORS.MaxAge == 0 {
		c.CORS.MaxAge = DefaultCORSMaxAge
	}
}

func (c *Config) Validate() error {
	v := validator.New()

	_ = v.RegisterValidation("startswith", validateStartsWith)

	if err := v.Struct(c); err != nil {
		return err
	}

	if c.Proto == "https" && (c.TLS.Cert == "" || c.TLS.Key == "") {
		return fmt.Errorf("tls.cert and tls.key are required for https")
	}

	for _, g := range c.Gateways {
		if g.Auth && c.AuthConfig.Key == "" {
			return fmt.Errorf("auth.key is required when auth=true in gateway %s", g.Prefix)
		}
	}

	return nil
}
