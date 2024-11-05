package cmds

import (
	"fmt"

	"github.com/Layr-Labs/avs-devnet/src/cmds/flags"
	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/urfave/cli/v2"
)

func Start(ctx *cli.Context) error {
	fmt.Println("Starting devnet...")
	pkgName := ctx.String(flags.KurtosisPackageFlag.Name)
	configPath, devnetName, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	if !fileExists(configPath) {
		return cli.Exit("Config file doesn't exist: "+configPath, 2)
	}

	config, err := config.LoadFromPath(configPath)
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
