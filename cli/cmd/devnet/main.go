package main

import (
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
		Name:   "start",
		Usage:  "Start the devnet",
		Flags:  []cli.Flag{},
		Action: start,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:   "stop",
		Usage:  "Stop the devnet",
		Flags:  []cli.Flag{},
		Action: stop,
	})

	app.Run(os.Args)
}

func start(ctx *cli.Context) error {
	argsFile := ctx.Args().First()
	devnetName := nameFromArgsFile(argsFile)

	return kurtosisRun("run", "../kurtosis_package/", "--enclave", devnetName, "--args-file", argsFile)
}

func stop(ctx *cli.Context) error {
	argsFile := ctx.Args().First()
	devnetName := nameFromArgsFile(argsFile)

	return kurtosisRun("enclave", "rm", "-f", devnetName)
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
