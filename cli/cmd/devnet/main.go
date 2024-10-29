package main

import (
	"os"
	"os/exec"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

var version = "development"

var configFlag = cli.StringFlag{
	Name:    "config",
	Aliases: []string{"c"},
	Usage:   "Load devnet configuration from `FILE`",
}

func main() {
	app := cli.NewApp()
	app.Name = "devnet"
	app.Usage = "start an AVS development network"
	app.Version = version

	app.Commands = append(app.Commands, &cli.Command{
		Name:   "run",
		Flags:  []cli.Flag{&configFlag},
		Action: run,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:   "clean",
		Flags:  []cli.Flag{&configFlag},
		Action: clean,
	})

	app.Run(os.Args)
}

func run(ctx *cli.Context) error {
	argsFile := ctx.String(configFlag.Name)
	devnetName, err := getDevnetName(argsFile)
	if err != nil {
		return err
	}

	cmd := exec.Command("kurtosis", "run", "../kurtosis_package/", "--enclave", devnetName, "--args-file", argsFile)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func clean(ctx *cli.Context) error {
	argsFile := ctx.String(configFlag.Name)
	devnetName, err := getDevnetName(argsFile)
	if err != nil {
		return err
	}
	cmd := exec.Command("kurtosis", "enclave", "rm", "-f", devnetName)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func getDevnetName(filePath string) (string, error) {
	if filePath == "" {
		return "devnet", nil
	}
	file, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	var devnetConfig struct {
		Name *string
	}
	err = yaml.Unmarshal(file, &devnetConfig)
	if devnetConfig.Name == nil {
		return "devnet", err
	}
	return *devnetConfig.Name, err
}
