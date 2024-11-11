package config

import (
	_ "embed"
	"os"

	"gopkg.in/yaml.v3"
)

type Deployment struct {
	Name string `yaml:"name"`
	Repo string `yaml:"repo"`
	Ref  string `yaml:"ref"`
	// non-exhaustive
}

type Service struct {
	Name         string  `yaml:"name"`
	Image        string  `yaml:"image"`
	BuildContext *string `yaml:"build_context"`
	BuildFile    *string `yaml:"build_file"`
	// non-exhaustive
}

type DevnetConfig struct {
	Deployments []Deployment `yaml:"deployments"`
	Services    []Service    `yaml:"services"`
	// non-exhaustive
}

func LoadFromPath(filePath string) (DevnetConfig, error) {
	var config DevnetConfig
	file, err := os.ReadFile(filePath)
	if err != nil {
		return config, err
	}
	return Unmarshal(file)
}

func Unmarshal(file []byte) (DevnetConfig, error) {
	var config DevnetConfig
	err := yaml.Unmarshal(file, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

//go:embed default_config.yaml
var DefaultConfig string
