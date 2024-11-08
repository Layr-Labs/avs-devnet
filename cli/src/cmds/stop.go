package cmds

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func Stop(ctx *cli.Context) error {
	_, devnetName, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	fmt.Println("Stopping devnet...")
	return kurtosisRun("enclave", "rm", "-f", devnetName)
}
