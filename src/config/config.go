package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Deployment struct {
	// Name for the deployment
	Name string `yaml:"name"`
	// URL to the git repo containing the contracts (can be local)
	Repo string `yaml:"repo"`
	// The git ref to checkout
	Ref string `yaml:"ref"`
	// Path to the contracts dir
	ContractsPath string `yaml:"contracts_path"`
	// Path to the deployment script, relative to the contracts directory
	Script string `yaml:"script"`

	// non-exhaustive
}

// Returns the path to the deployment script (i.e. `Script`) but without the trailing contract name
// Example: "contracts/contracts.sol:Contract" -> "contracts/contracts.sol"
func (d Deployment) GetScriptPath() string {
	scriptPath := strings.SplitAfterN(d.Script, ".sol:", 1)[0]
	return strings.TrimSuffix(scriptPath, ":")
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
