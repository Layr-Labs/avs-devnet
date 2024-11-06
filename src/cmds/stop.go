package cmds

import (
	"fmt"

	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
	"github.com/urfave/cli/v2"
)

func Stop(ctx *cli.Context) error {
	_, devnetName, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	fmt.Println("Stopping devnet...")
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	if err != nil {
		return cli.Exit(err, 2)
	}
	if err = kurtosisCtx.DestroyEnclave(ctx.Context, devnetName); err != nil {
		return cli.Exit(err, 3)
	}
	return err
}
