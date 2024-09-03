package config

import (
	"api/internal/lib/sl"
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

const (
	EnvLocal       = "local"
	EnvDevelopment = "dev"
	EnvProduction  = "prod"
)

type Config struct {
	Env      string   `yaml:"env" env-required:"true"`
	Server   Server   `yaml:"server" env-required:"true"`
	Mail     Mail     `yaml:"mail" env-required:"true"`
	Token    Token    `yaml:"token" env-required:"true"`
	Postgres Postgres `yaml:"postgres" env-required:"true"`
	Redis    Redis    `yaml:"redis" env-required:"true"`
}

type Server struct {
	Address     string        `yaml:"address" env-required:"true"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

type Mail struct {
	Address  string `yaml:"address" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
	SMTP     SMTP   `yaml:"smtp" env-required:"true"`
}

type SMTP struct {
	Address string `yaml:"address" env-required:"true"`
	Port    string `yaml:"port" env-required:"true"`
}

type Token struct {
	Secret string `yaml:"secret" env-required:"true"`
}

type Postgres struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Name     string `yaml:"name"`
	Password string `yaml:"password"`
	ModeSSL  string `yaml:"sslmode"`
}

type Redis struct {
	Addr     string `yaml:"addr" env-required:"true"`
	Password string `yaml:"password" env-required:"true"`
}

// MustLoad loads config to a new Config instance and return it
func MustLoad() *Config {
	_ = godotenv.Load()

	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		slog.Error("missed CONFIG_PATH parameter")
		os.Exit(1)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		slog.Error("config file does not exist", slog.String("path", configPath))
		os.Exit(1)
	}

	var config Config

	if err := cleanenv.ReadConfig(configPath, &config); err != nil {
		slog.Error("cannot read config", sl.Err(err))
		os.Exit(1)
	}

	return &config
}
