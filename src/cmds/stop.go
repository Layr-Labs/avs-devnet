package cmds

import (
	"fmt"

	"github.com/Layr-Labs/avs-devnet/src/kurtosis"
	"github.com/urfave/cli/v2"
)

// Stops the devnet with the given context
func Stop(ctx *cli.Context) error {
	devnetName, _, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	fmt.Println("Stopping devnet...")
	kurtosisCtx, err := kurtosis.InitKurtosisContext()
	if err != nil {
		return cli.Exit(err, 2)
	}
	if !kurtosisCtx.EnclaveExists(ctx.Context, devnetName) {
		return cli.Exit("Failed to find '"+devnetName+"'. Maybe it's not running?", 3)
	}
	if err = kurtosisCtx.DestroyEnclave(ctx.Context, devnetName); err != nil {
		return cli.Exit(err, 4)
	}
	fmt.Println("Devnet stopped!")
	return nil
}
