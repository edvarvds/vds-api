package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server struct {
		Port         string   `yaml:"port"`
		RateLimit    int      `yaml:"rate_limit"`
		AdminToken   string   `yaml:"admin_token"`
		AllowedHosts []string `yaml:"allowed_hosts"`
		AllowedIPs   []string `yaml:"allowed_ips"`
	} `yaml:"server"`
	Redis struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Password string `yaml:"password"`
		Username string `yaml:"username"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
	API struct {
		CPFEndpoint string `yaml:"cpf_endpoint"`
		Token       string `yaml:"token"`
	} `yaml:"api"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return &config, nil
} 