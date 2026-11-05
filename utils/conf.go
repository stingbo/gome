package utils

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

type MeConfig struct {
	Debug     bool     `yaml:"Debug"`
	LogLevel  string   `yaml:"LogLevel"`
	MQconf    RabbitMQ `yaml:"rabbitmq"`
	CacheConf Cache    `yaml:"redis"`
	GRPCConf  GRPC     `yaml:"grpc"`
	MeConf    Mengine  `yaml:"gome"`
}

type Cache struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
}

type RabbitMQ struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type GRPC struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type Mengine struct {
	Accuracy int `yaml:"accuracy"`
}

// LoadConfig reads configuration at the executable boundary.
func LoadConfig() (*MeConfig, error) {
	path := os.Getenv("GOME_CONFIG")
	if path == "" {
		path = "config.yaml"
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read configuration: %w", err)
	}
	var config MeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse configuration %s: %w", path, err)
	}
	if config.MeConf.Accuracy < 0 || config.MeConf.Accuracy > 15 {
		return nil, fmt.Errorf("gome.accuracy must be between 0 and 15")
	}
	return &config, nil
}
