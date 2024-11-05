package main

import (
	"errors"
	"fmt"
	"log"
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

type Service struct {
	Name         string  `yaml:"name"`
	Image        string  `yaml:"image"`
	BuildContext *string `yaml:"build_context"`
	BuildFile    *string `yaml:"build_file"`
}

type DevnetConfig struct {
	Deployments []Deployment `yaml:"deployments"`
	Services    []Service    `yaml:"services"`
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

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
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
`

func InitCmd(ctx *cli.Context) error {
	configFileName, _, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	if fileExists(configFileName) {
		return cli.Exit("Config file already exists: "+configFileName, 2)
	}

	fmt.Println("Creating new devnet configuration file in", configFileName)

	file, err := os.Create(configFileName)
	if err != nil {
		return cli.Exit(err, 3)
	}
	_, err = file.WriteString(DefaultConfig)
	if err != nil {
		return cli.Exit(err, 4)
	}
	return file.Close()
}

func StartCmd(ctx *cli.Context) error {
	fmt.Println("Starting devnet...")
	pkgName := ctx.String(packageNameFlag.Name)
	configPath, devnetName, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	if !fileExists(configPath) {
		return cli.Exit("Config file doesn't exist: "+configPath, 2)
	}

	config, err := loadConfigFile(configPath)
	if err != nil {
		return cli.Exit(err, 2)
	}

	err = buildDockerImages(config)
	if err != nil {
		return cli.Exit(err, 3)
	}

	if err = kurtosisRun("enclave", "add", "--name", devnetName); err != nil {
		return cli.Exit(err, 4)
	}

	err = uploadLocalRepos(config, devnetName)
	if err != nil {
		return cli.Exit(err, 5)
	}

	return kurtosisRun("run", pkgName, "--enclave", devnetName, "--args-file", configPath)
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
	configFileName := ctx.String(configFileNameFlag.Name)
	devnetName, err := nameFromConfigFile(configFileName)
	if err != nil {
		return cli.Exit(err, 1)
	}

	err = printAddresses(args.Slice(), devnetName)

	if err != nil {
		return cli.Exit(err, 2)
	}
	return nil
}

func printAddresses(args []string, devnetName string) error {
	failed := false
	cacheDir, err := os.MkdirTemp(os.TempDir(), ".devnet_cache")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(cacheDir) }()
	cached := make(map[string]string)

	for _, arg := range args {
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
		output, ok := readArtifact(file, contractName)
		if !ok {
			fmt.Println("Error getting", arg)
			failed = true
			continue
		}
		fmt.Println(strings.TrimSpace(output))
	}
	if failed {
		return errors.New("failed to get some addresses")
	}
	return nil
}

func readArtifact(file string, contractName string) (string, bool) {
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
	if !res.Exists() {
		return "", false
	}
	return res.String(), true
}

func uploadLocalRepos(config DevnetConfig, devnetName string) error {
	alreadyUploaded := make(map[string]bool)

	for _, deployment := range config.Deployments {
		if deployment.Repo == "" {
			continue
		}
		repoUrl, err := url.Parse(deployment.Repo)
		if err != nil {
			return err
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
			return err
		}
		alreadyUploaded[repoUrl.Path] = true
	}
	return nil
}

func buildDockerImages(config DevnetConfig) error {
	errChan := make(chan error)
	numBuilds := 0
	for _, service := range config.Services {
		if service.BuildContext == nil {
			continue
		}
		image := service.Image
		cmdArgs := []string{"docker", "build", *service.BuildContext, "-t", image}
		if service.BuildFile != nil {
			cmdArgs = append(cmdArgs, "-f", *service.BuildFile)
		}
		cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
		fmt.Println("Building image", image)
		go func() {
			output, err := cmd.CombinedOutput()
			if err != nil {
				err = fmt.Errorf("building image '%s' failed: %w\n%s", image, err, output)
			}
			errChan <- err
		}()
		numBuilds += 1
	}
	errs := make([]error, numBuilds)
	for range numBuilds {
		errs = append(errs, <-errChan)
	}
	return errors.Join(errs...)
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
	configFileName := args.First()
	var devnetName string
	if configFileName == "" {
		configFileName = "devnet.yaml"
		devnetName = "devnet"
	} else {
		name, err := nameFromConfigFile(configFileName)
		if err != nil {
			return "", "", err
		}
		devnetName = name
	}
	return configFileName, devnetName, nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func nameFromConfigFile(fileName string) (string, error) {
	name := filepath.Base(fileName)
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

func loadConfigFile(fileName string) (DevnetConfig, error) {
	var config DevnetConfig
	file, err := os.ReadFile(fileName)
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
