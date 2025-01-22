package main

import (
	"log"
	"os"

	"github.com/Layr-Labs/avs-devnet/src/cmds"
	"github.com/Layr-Labs/avs-devnet/src/cmds/flags"
	"github.com/urfave/cli/v2"
)

var version = "development"

func main() {
	app := cli.NewApp()
	app.Name = "avs-devnet"
	app.Usage = "start an AVS devnet"
	app.Version = version
	app.Flags = append(app.Flags, &flags.KurtosisPackageFlag)

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "init",
		Usage:     "Initialize a devnet configuration file",
		Args:      true,
		ArgsUsage: "[<config-file>]",
		Flags:     []cli.Flag{},
		Action:    cmds.InitCmd,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "start",
		Usage:     "Start devnet from configuration file",
		Args:      true,
		ArgsUsage: "[<config-file>]",
		Flags:     []cli.Flag{},
		Action:    cmds.StartCmd,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "stop",
		Usage:     "Stop devnet from configuration file",
		Args:      true,
		ArgsUsage: "[<config-file>]",
		Flags:     []cli.Flag{},
		Action:    cmds.StopCmd,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "get-address",
		Usage:     "Get a devnet contract or EOA address",
		Args:      true,
		ArgsUsage: "<contract-name>...",
		Flags:     []cli.Flag{&flags.ConfigFilePathFlag},
		Action:    cmds.GetAddress,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "get-ports",
		Usage:     "Get the published ports on the devnet",
		Args:      true,
		ArgsUsage: "[<config-file>]",
		Flags:     []cli.Flag{},
		Action:    cmds.GetPorts,
	})

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
