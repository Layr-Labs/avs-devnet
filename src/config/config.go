package config

import (
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

const DefaultConfig = `deployments:
  # Deploy EigenLayer
  - type: EigenLayer
    ref: v0.4.2-mainnet-pepe
    # Whitelist a single strategy named MockETH, backed by a mock-token
    strategies: [MockETH]
    operators:
      # Register a single operator with EigenLayer
      - name: operator1
        keys: operator1_ecdsa
        # Deposit 1e17 tokens into the MockETH strategy
        strategies:
          MockETH: 100000000000000000

# Specify keys to generate
keys:
  - name: operator1_ecdsa
    type: ecdsa
  - name: operator1_bls
    type: bls

# ethereum-package configuration
ethereum_package:
  participants:
    - el_type: erigon
  additional_services:
    - blockscout
`
