package cmds

import (
	"github.com/Layr-Labs/avs-devnet/src/cmds/flags"
	"github.com/urfave/cli/v2"
)

func NewCliApp() *cli.App {
	app := cli.NewApp()
	app.Name = "devnet"
	app.Usage = "start an AVS devnet"
	app.Version = Version
	app.Flags = append(app.Flags, &flags.KurtosisPackageFlag)

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "init",
		Usage:     "Initialize a devnet configuration file",
		Args:      true,
		ArgsUsage: "[<config-file>]",
		Flags:     []cli.Flag{},
		Action:    Init,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "start",
		Usage:     "Start devnet from configuration file",
		Args:      true,
		ArgsUsage: "[<config-file>]",
		Flags:     []cli.Flag{},
		Action:    Start,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "stop",
		Usage:     "Stop devnet from configuration file",
		Args:      true,
		ArgsUsage: "[<config-file>]",
		Flags:     []cli.Flag{},
		Action:    Stop,
	})

	app.Commands = append(app.Commands, &cli.Command{
		Name:      "get-address",
		Usage:     "Get a devnet contract or EOA address",
		Args:      true,
		ArgsUsage: "<contract-name>...",
		Flags:     []cli.Flag{&flags.ConfigFilePathFlag},
		Action:    GetAddress,
	})
	return app
}
