package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	"github.com/tidwall/gjson"
)

var version = "development"

type Deployment struct {
	Name string `yaml:"name"`
	Repo string `yaml:"repo"`
	Ref  string `yaml:"ref"`
	// non-exhaustive
}

type DevnetConfig struct {
	Deployments []Deployment `yaml:"deployments"`
	// non-exhaustive
}

var packageNameFlag = cli.StringFlag{
	Name:    "package-name",
	Usage:   "Locator for the Kurtosis package to run",
	Hidden:  true,
	EnvVars: []string{"AVS_DEVNET__PACKAGE_NAME"},
	Value:   "github.com/Layr-Labs/avs-devnet",
}

var configFileNameFlag = cli.StringFlag{
	Name:  "config-file",
	Usage: "Path to the devnet configuration file",
	Value: "devnet.yaml",
}

func main() {
	app := cli.NewApp()
	app.Name = "devnet"
	app.Usage = "start an AVS devnet"
	app.Version = version
	app.Flags = append(app.Flags, &packageNameFlag)

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "init",
		Usage:     "Initialize a devnet configuration file",
		Args:      true,
		ArgsUsage: "[<config-file>]",
		Flags:     []cli.Flag{},
		Action:    InitCmd,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "start",
		Usage:     "Start devnet from configuration file",
		Args:      true,
		ArgsUsage: "[<config-file>]",
		Flags:     []cli.Flag{},
		Action:    StartCmd,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "stop",
		Usage:     "Stop devnet from configuration file",
		Args:      true,
		ArgsUsage: "[<config-file>]",
		Flags:     []cli.Flag{},
		Action:    StopCmd,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "get-address",
		Usage:     "Get a devnet contract or EOA address",
		Args:      true,
		ArgsUsage: "<contract-name>...",
		Flags:     []cli.Flag{&configFileNameFlag},
		Action:    GetAddressCmd,
	})

	app.Run(os.Args)
}

var DefaultConfig = `deployments:
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
    - dora
`

func InitCmd(ctx *cli.Context) error {
	argsFile, _, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	if fileExists(argsFile) {
		return cli.Exit("Config file already exists: "+argsFile, 2)
	}

	fmt.Println("Creating new devnet configuration file in", argsFile)

	file, err := os.Create(argsFile)
	if err != nil {
		return cli.Exit(err, 3)
	}
	file.WriteString(DefaultConfig)
	return file.Close()
}

func StartCmd(ctx *cli.Context) error {
	fmt.Println("Starting devnet...")
	pkgName := ctx.String(packageNameFlag.Name)
	argsFile, devnetName, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	if !fileExists(argsFile) {
		return cli.Exit("Config file doesn't exist: "+argsFile, 2)
	}

	config, err := loadArgsFile(argsFile)
	if err != nil {
		return cli.Exit(err, 2)
	}

	if err := kurtosisRun("enclave", "add", "--name", devnetName); err != nil {
		return cli.Exit(err, 5)
	}

	alreadyUploaded := make(map[string]bool)

	for _, deployment := range config.Deployments {
		if deployment.Repo == "" {
			continue
		}
		repoUrl, err := url.Parse(deployment.Repo)
		if err != nil {
			return cli.Exit(err, 6)
		}
		if repoUrl.Scheme != "file" && repoUrl.Scheme != "" {
			continue
		}
		if alreadyUploaded[repoUrl.Path] {
			continue
		}
		path := repoUrl.Path
		// Upload the file with the path as the name
		if err := kurtosisRun("files", "upload", "--name", path, devnetName, path); err != nil {
			return cli.Exit(err, 7)
		}
		alreadyUploaded[repoUrl.Path] = true
	}

	return kurtosisRun("run", pkgName, "--enclave", devnetName, "--args-file", argsFile)
}

func StopCmd(ctx *cli.Context) error {
	_, devnetName, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	fmt.Println("Stopping devnet...")
	return kurtosisRun("enclave", "rm", "-f", devnetName)
}

func GetAddressCmd(ctx *cli.Context) error {
	args := ctx.Args()
	argsFile := ctx.String(configFileNameFlag.Name)
	devnetName, err := nameFromArgsFile(argsFile)
	if err != nil {
		return cli.Exit(err, 1)
	}

	failed := false

	cacheDir, err := os.MkdirTemp(os.TempDir(), ".devnet_cache")
	if err != nil {
		return cli.Exit(err, 2)
	}
	defer func() { _ = os.RemoveAll(cacheDir) }()

	cached := make(map[string]string)

	for _, arg := range args.Slice() {
		// contract name is like "artifact-name:contract-name"
		path := strings.Split(arg, ":")
		// TODO: assume a length of 1 means it's just the contract name
		if len(path) > 2 || len(path) == 1 {
			fmt.Println("Invalid contract name: " + arg)
			failed = true
			continue
		}
		artifactName := path[0]
		contractName := path[1]
		file, ok := cached[artifactName]
		if !ok {
			readFile, err := readJsonArtifact(cacheDir, devnetName, artifactName)
			if err != nil {
				fmt.Println("Error reading artifact", artifactName+":", err)
				failed = true
				readFile = ""
			}
			file = string(readFile)
			cached[artifactName] = file
		}
		var jsonPath string
		if strings.HasPrefix(contractName, ".") {
			// This uses the absolute path
			jsonPath = "addresses" + contractName + "|@pretty"
		} else if contractName != "" {
			// This searches for `contractName` inside the json
			// Since there are multiple results, `|0` is used to get the first one
			jsonPath = "@dig:" + contractName + "|0|@pretty"
		} else {
			// This just prints the whole json
			jsonPath = "@pretty"
		}
		res := gjson.Get(string(file), jsonPath)
		if res.Exists() {
			fmt.Print(res.String())
		} else {
			fmt.Println("Contract not found: " + arg)
			failed = true
		}
	}
	if failed {
		return cli.Exit("", 7)
	}
	return nil
}

func readJsonArtifact(cacheDir string, devnetName string, artifactName string) (string, error) {
	err := exec.Command("sh", "-c", "cd "+cacheDir+" && kurtosis files download "+devnetName+" "+artifactName).Run()
	if err != nil {
		return "", err
	}
	artifactPath := filepath.Join(cacheDir, artifactName)
	dirEntry, err := os.ReadDir(artifactPath)
	if err != nil {
		return "", err
	}
	var readFile []byte
	for _, entry := range dirEntry {
		if entry.IsDir() {
			continue
		}
		fileName := entry.Name()
		if filepath.Ext(fileName) != ".json" {
			continue
		}
		readFile, err = os.ReadFile(filepath.Join(artifactPath, fileName))
		if err != nil {
			return "", err
		}
		break
	}
	return string(readFile), nil
}

func parseArgs(ctx *cli.Context) (string, string, error) {
	args := ctx.Args()
	if args.Len() > 1 {
		return "", "", errors.New("expected exactly 1 argument: <config-file>")
	}
	argsFile := args.First()
	var devnetName string
	if argsFile == "" {
		argsFile = "devnet.yaml"
		devnetName = "devnet"
	} else {
		name, err := nameFromArgsFile(argsFile)
		if err != nil {
			return "", "", err
		}
		devnetName = name
	}
	return argsFile, devnetName, nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func nameFromArgsFile(argsFile string) (string, error) {
	name := filepath.Base(argsFile)
	name = strings.Split(name, ".")[0]
	name = strings.ReplaceAll(name, "_", "-")
	matches, err := regexp.MatchString("^[-A-Za-z0-9]{1,60}$", name)
	if err != nil {
		// Error in regex pattern
		panic(err)
	}
	if !matches {
		return "", errors.New("Invalid devnet name: " + name)
	}
	return name, nil
}

func loadArgsFile(argsFile string) (DevnetConfig, error) {
	var config DevnetConfig
	file, err := os.ReadFile(argsFile)
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func kurtosisRun(args ...string) error {
	cmd := exec.Command("kurtosis", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
