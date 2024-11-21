package config

import (
	_ "embed"
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
	scriptPath := strings.SplitAfterN(d.Script, ".sol:", 2)[0]
	return strings.TrimSuffix(scriptPath, ":")
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

//go:embed default_config.yaml
var DefaultConfig string
