package config

import (
	"errors"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"os"
)

type BaseConfig struct {
	AppName          string         `json:"app_name" yaml:"app_name"`
	AppTokens        []string       `json:"app_tokens" yaml:"app_tokens" env:"app_tokens"` // comma separated strings
	ServerConfig     ServerConfig   `json:"server_config" yaml:"server_config"`
	GatewayConfig    GatewayConfig  `yaml:"gateway_config" json:"gateway_config"`
	DatabaseConfig   DatabaseConfig `json:"database_config" yaml:"database_config"`
	LogLevel         string         `yaml:"log_level" json:"log_level"`
	TraceProviderUrl string         `yaml:"trace_provider_url" json:"trace_provider_url"`
}

type ServerConfig struct {
	Address          string `yaml:"string" json:"address"`
	Port             string `yaml:"port" json:"port"`
	KeepAliveTime    uint   `yaml:"keep_alive_time" json:"keep_alive_time"`
	KeepAliveTimeOut uint   `yaml:"keep_alive_time_out" json:"keep_alive_time_out"`
}

type GatewayConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	Address     string `yaml:"gateway_address" json:"gateway_address"`
	Port        uint32    `yaml:"port" json:"port"`
	Url         string `yaml:"url" json:"url"`
	SwaggerFile string `yaml:"swagger_file" json:"swagger_file"`
	Endpoint    string `yaml:"endpoint" json:"endpoint"`
}

type DatabaseConfig struct {
	Type         string `yaml:"type" json:"type"`
	HostName     string `yaml:"host_name" json:"host_name"`
	Port         string `yaml:"port" json:"port"`
	UserName     string `yaml:"user_name" json:"user_name"`
	DatabaseName string `yaml:"database_name" json:"database_name"`
	Password     string `yaml:"password" json:"password" envconfig:"DATABASE_PASSWORD"`
}

func ReadFile(filePath string, cfg interface{}) error {
	path, found := os.LookupEnv(filePath)
	if !found {
		return errors.New("config file not found")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	ReadEnv(cfg)
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		return err
	}
	return nil
}

func ReadEnv(cfg interface{}) error {
	err := envconfig.Process("", cfg)
	if err != nil {
		return err
	}
	return nil
}
