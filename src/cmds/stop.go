package cmds

import (
	"context"
	"errors"
	"fmt"

	"github.com/Layr-Labs/avs-devnet/src/kurtosis"
	"github.com/urfave/cli/v2"
)

var ErrEnclaveNotExists = errors.New("enclave doesn't exist")

// Stops the devnet with the given context.
func StopCmd(ctx *cli.Context) error {
	devnetName, _, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	fmt.Println("Stopping devnet...")
	err = Stop(ctx.Context, devnetName)
	if err == ErrEnclaveNotExists {
		return cli.Exit("Failed to find '"+devnetName+"'. Maybe it's not running?", 2)
	} else if err != nil {
		return cli.Exit(err, 3)
	}
	fmt.Println("Devnet stopped!")
	return nil
}

func Stop(ctx context.Context, devnetName string) error {
	kurtosisCtx, err := kurtosis.InitKurtosisContext()
	if err != nil {
		return fmt.Errorf("failed to initialize kurtosis context: %w", err)
	}
	if !kurtosisCtx.EnclaveExists(ctx, devnetName) {
		return ErrEnclaveNotExists
	}
	err = kurtosisCtx.DestroyEnclave(ctx, devnetName)
	if err != nil {
		return fmt.Errorf("failed to destroy enclave: %w", err)
	}
	return nil
}
