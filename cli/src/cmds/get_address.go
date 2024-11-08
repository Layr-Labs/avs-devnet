package cmds

import (
	"github.com/Layr-Labs/avs-devnet/src/cmds/flags"
	"github.com/urfave/cli/v2"
)

func GetAddress(ctx *cli.Context) error {
	args := ctx.Args()
	configFileName := ctx.String(flags.ConfigFilePathFlag.Name)
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
