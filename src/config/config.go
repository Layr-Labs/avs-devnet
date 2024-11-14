package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Deployment struct {
	Name          string `yaml:"name"`
	Repo          string `yaml:"repo"`
	Ref           string `yaml:"ref"`
	ContractsPath string `yaml:"contracts_path"`
	Script        string `yaml:"script"`
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
