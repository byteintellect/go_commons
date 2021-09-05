package config

import (
	"errors"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"os"
)

type BaseConfig struct {
	AppName   string   `json:"app_name" yaml:"app_name"`
	AppTokens []string `json:"app_tokens" yaml:"app_tokens" env:"app_tokens"` // comma separated strings
	Address   string   `json:"address" yaml:"address"`
	Port      uint64   `json:"port" yaml:"port"`
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
