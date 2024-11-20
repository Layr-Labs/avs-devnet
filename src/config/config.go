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
	// The service name
	Name string `yaml:"name"`
	// The docker image's name
	Image string `yaml:"image"`
	// Optional. A custom build command to run to build the docker image.
	// Ignored if BuildContext is set.
	BuildCmd *string `yaml:"build_cmd"`
	// Optional. The build context to use to build the docker image.
	BuildContext *string `yaml:"build_context"`
	// Optional. The build file to specify when building the docker image.
	// Ignored unless BuildContext is set.
	BuildFile *string `yaml:"build_file"`
	// non-exhaustive
}

// A devnet specification
type DevnetConfig struct {
	// Contains contract groups to deploy
	Deployments []Deployment `yaml:"deployments"`
	// Contains off-chain services to start
	Services []Service `yaml:"services"`
	// non-exhaustive
	raw []byte
}

// Loads a DevnetConfig from a file
func LoadFromPath(filePath string) (DevnetConfig, error) {
	var config DevnetConfig
	file, err := os.ReadFile(filePath)
	if err != nil {
		return config, err
	}
	return Unmarshal(file)
}

// Loads a DevnetConfig from a byte slice
func Unmarshal(file []byte) (DevnetConfig, error) {
	var config DevnetConfig
	config.raw = file
	err := yaml.Unmarshal(file, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

// Serializes the config.
// If read from a file, the serialized config will be the same as the file's content.
func (c DevnetConfig) Marshal() []byte {
	return c.raw
}

//go:embed default_config.yaml
var DefaultConfig string
