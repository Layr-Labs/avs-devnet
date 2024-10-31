package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

var version = "development"

func main() {
	app := cli.NewApp()
	app.Name = "devnet"
	app.Usage = "start an AVS devnet"
	app.Version = version

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
        keystore: operator1_ecdsa_keystore
        # Deposit 1e17 tokens into the MockETH strategy
        strategies:
          MockETH: 100000000000000000

# Specify keys to generate
keystores:
  - name: operator1_ecdsa_keystore
    type: ecdsa
  - name: operator1_bls_keystore
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
		return err
	}
	if fileExists(argsFile) {
		return cli.Exit("Config file already exists: "+argsFile, 2)
	}
	file, err := os.Create(argsFile)
	if err != nil {
		return err
	}
	file.WriteString(DefaultConfig)
	return file.Close()
}

func StartCmd(ctx *cli.Context) error {
	argsFile, devnetName, err := parseArgs(ctx)
	if err != nil {
		return err
	}
	if !fileExists(argsFile) {
		return cli.Exit("Config file doesn't exist: "+argsFile, 2)
	}

	return kurtosisRun("run", "../kurtosis_package/", "--enclave", devnetName, "--args-file", argsFile)
}

func StopCmd(ctx *cli.Context) error {
	_, devnetName, err := parseArgs(ctx)
	if err != nil {
		return err
	}
	return kurtosisRun("enclave", "rm", "-f", devnetName)
}

func parseArgs(ctx *cli.Context) (string, string, error) {
	args := ctx.Args()
	if args.Len() > 1 {
		return "", "", cli.Exit("Expected exactly 1 argument: <config-file>", 1)
	}
	argsFile := args.First()
	var devnetName string
	if argsFile == "" {
		argsFile = "devnet.yaml"
		devnetName = "devnet"
	} else {
		devnetName = nameFromArgsFile(argsFile)
	}
	return argsFile, devnetName, nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func nameFromArgsFile(argsFile string) string {
	base := filepath.Base(argsFile)
	return strings.Split(base, ".")[0]
}

func kurtosisRun(args ...string) error {
	cmd := exec.Command("kurtosis", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
