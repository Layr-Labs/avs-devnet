package config

import (
	_ "embed"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// A devnet specification.
type DevnetConfig struct {
	// Contains contract groups to deploy
	Deployments []Deployment `yaml:"deployments"`
	// Contains off-chain services to start
	Services []Service `yaml:"services"`
	// Contains artifacts to generate
	// The key is the artifact name
	Artifacts map[string]Artifact `yaml:"artifacts"`
	
	// 🆕 Add this
	EthereumPackage *EthereumPackageConfig `yaml:"ethereum_package"`

	// non-exhaustive

	raw []byte
}

type EthereumPackageConfig struct {
	Participants     []EthereumParticipant `yaml:"participants"`
	AdditionalSvcs   []string              `yaml:"additional_services"`
	NetworkParams    map[string]interface{} `yaml:"network_params"`
}

type EthereumParticipant struct {
	ELType  string `yaml:"el_type"`
	ELImage string `yaml:"el_image,omitempty"`
}


// A group of contracts to deploy.
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
// Example: "contracts/contracts.sol:Contract" -> "contracts/contracts.sol".
func (d Deployment) GetScriptPath() string {
	const maxSplits = 2
	scriptPath := strings.SplitAfterN(d.Script, ".sol:", maxSplits)[0]
	return strings.TrimSuffix(scriptPath, ":")
}

// A service to start.
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

// An artifact to generate.
// The key is the file name, and the value is the file's definition.
type Artifact struct {
	// Contains a mapping with the files to generate
	Files map[string]ArtifactFile `yaml:"files"`

	// non-exhaustive
}

// The definition of an artifact file.
// Must be either a static file or a template.
type ArtifactFile struct {
	// URL to a file to upload to the enclave
	StaticFile *string `yaml:"static_file"`
	// Content of the file, with optional templates
	Template *string `yaml:"template"`
}

// Loads a DevnetConfig from a file.
func LoadFromPath(filePath string) (DevnetConfig, error) {
	var config DevnetConfig
	file, err := os.ReadFile(filePath)
	if err != nil {
		return config, err
	}
	return Unmarshal(file)
}

// Loads a DevnetConfig from a byte slice.
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
var defaultConfig []byte

func DefaultConfig() DevnetConfig {
	cfg, _ := Unmarshal(defaultConfig)
	return cfg
}

func DefaultConfigStr() string {
	return string(defaultConfig)
}
