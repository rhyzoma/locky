package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Environment string         `yaml:"environment"`
	Server      ServerConfig   `yaml:"server"`
	Database    DatabaseConfig `yaml:"database"`
	Mail        MailConfig     `yaml:"mail"`
	Security    SecurityConfig `yaml:"security"`
	Locky       LockyConfig    `yaml:"locky"`
}

type ServerConfig struct {
	GRPCAddr string    `yaml:"grpc_addr"`
	TLS      TLSConfig `yaml:"tls"`
}

type DatabaseConfig struct {
	DSN      string `yaml:"dsn"`
	MaxConns int32  `yaml:"max_conns"`
}

type MailConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	From      string `yaml:"from"`
	VerifyURL string `yaml:"verify_url"`
}

type TLSConfig struct {
	CertFile          string `yaml:"cert_file"`
	KeyFile           string `yaml:"key_file"`
	ClientCAFile      string `yaml:"client_ca_file"`
	RequireClientCert bool   `yaml:"require_client_cert"`
}

type SecurityConfig struct {
	JWTSigningKey   string   `yaml:"jwt_signing_key"`
	JWTKeyID        string   `yaml:"jwt_key_id"`
	LoginRateLimit  int      `yaml:"login_rate_limit"`
	LoginRateWindow Duration `yaml:"login_rate_window"`
}

type LockyConfig struct {
	AutoVerifyEmail bool `yaml:"auto_verify_email"`
}

type Duration time.Duration

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var value string
	if err := node.Decode(&value); err != nil {
		return fmt.Errorf("duration must be a string such as 2s: %w", err)
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", value, err)
	}
	*d = Duration(parsed)
	return nil
}

func Load(path string) (Config, error) {
	cfg := Config{
		Environment: "development",
		Server:      ServerConfig{GRPCAddr: ":8080"},
		Database:    DatabaseConfig{MaxConns: 20},
		Security:    SecurityConfig{JWTSigningKey: "local-development-key", JWTKeyID: "local-development", LoginRateLimit: 10, LoginRateWindow: Duration(time.Minute)},
		Mail:        MailConfig{Port: 587},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}
	if err := applyEnvironment(&cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyEnvironment(cfg *Config) error {
	if value := os.Getenv("APP_ENV"); value != "" {
		cfg.Environment = value
	}
	if value := os.Getenv("GRPC_ADDR"); value != "" {
		cfg.Server.GRPCAddr = value
	}
	if value := os.Getenv("TLS_CERT_FILE"); value != "" {
		cfg.Server.TLS.CertFile = value
	}
	if value := os.Getenv("TLS_KEY_FILE"); value != "" {
		cfg.Server.TLS.KeyFile = value
	}
	if value := os.Getenv("TLS_CLIENT_CA_FILE"); value != "" {
		cfg.Server.TLS.ClientCAFile = value
	}
	if value := os.Getenv("TLS_REQUIRE_CLIENT_CERT"); value != "" {
		require, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse TLS_REQUIRE_CLIENT_CERT: %w", err)
		}
		cfg.Server.TLS.RequireClientCert = require
	}
	if value := os.Getenv("JWT_SIGNING_KEY"); value != "" {
		cfg.Security.JWTSigningKey = value
	}
	if value := os.Getenv("JWT_KEY_ID"); value != "" {
		cfg.Security.JWTKeyID = value
	}
	if value := os.Getenv("LOGIN_RATE_LIMIT"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse LOGIN_RATE_LIMIT: %w", err)
		}
		cfg.Security.LoginRateLimit = limit
	}
	if value := os.Getenv("LOGIN_RATE_WINDOW"); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("parse LOGIN_RATE_WINDOW: %w", err)
		}
		cfg.Security.LoginRateWindow = Duration(duration)
	}
	if value := os.Getenv("DATABASE_URL"); value != "" {
		cfg.Database.DSN = value
	}
	if value := os.Getenv("DATABASE_MAX_CONNS"); value != "" {
		maxConns, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return fmt.Errorf("parse DATABASE_MAX_CONNS: %w", err)
		}
		cfg.Database.MaxConns = int32(maxConns)
	}
	if value := os.Getenv("MAIL_ENABLED"); value != "" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse MAIL_ENABLED: %w", err)
		}
		cfg.Mail.Enabled = enabled
	}
	if value := os.Getenv("SMTP_HOST"); value != "" {
		cfg.Mail.Host = value
	}
	if value := os.Getenv("SMTP_PORT"); value != "" {
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse SMTP_PORT: %w", err)
		}
		cfg.Mail.Port = port
	}
	if value := os.Getenv("SMTP_USERNAME"); value != "" {
		cfg.Mail.Username = value
	}
	if value := os.Getenv("SMTP_PASSWORD"); value != "" {
		cfg.Mail.Password = value
	}
	if value := os.Getenv("SMTP_FROM"); value != "" {
		cfg.Mail.From = value
	}
	if value := os.Getenv("VERIFY_URL"); value != "" {
		cfg.Mail.VerifyURL = value
	}
	if value := os.Getenv("AUTO_VERIFY_EMAIL"); value != "" {
		autoVerify, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("parse AUTO_VERIFY_EMAIL: %w", err)
		}
		cfg.Locky.AutoVerifyEmail = autoVerify
	}
	return nil
}

func (c Config) validate() error {
	if c.Server.GRPCAddr == "" {
		return errors.New("config: server grpc address is required")
	}
	if c.Security.JWTSigningKey == "" {
		return errors.New("config: jwt signing key is required")
	}
	if c.Security.LoginRateLimit <= 0 || time.Duration(c.Security.LoginRateWindow) <= 0 {
		return errors.New("config: login rate limit and window must be positive")
	}
	if c.Environment == "production" {
		if c.Database.DSN == "" {
			return errors.New("config: database dsn is required in production")
		}
		if c.Locky.AutoVerifyEmail {
			return errors.New("config: auto verify email is not allowed in production")
		}
		if c.Security.JWTSigningKey == "local-development-key" {
			return errors.New("config: development jwt signing key is not allowed in production")
		}
		if len(c.Security.JWTSigningKey) < 32 {
			return errors.New("config: jwt signing key must be at least 32 characters in production")
		}
		if c.Security.JWTKeyID == "" {
			return errors.New("config: jwt key id is required in production")
		}
		if c.Server.TLS.CertFile == "" || c.Server.TLS.KeyFile == "" {
			return errors.New("config: server tls certificate and key are required in production")
		}
		if !c.Mail.Enabled || c.Mail.Host == "" || c.Mail.From == "" || c.Mail.VerifyURL == "" {
			return errors.New("config: verification mail is required in production")
		}
	}
	return nil
}
