package kurtosis

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
)

type KurtosisCtx struct {
	*kurtosis_context.KurtosisContext
}

type EnclaveCtx struct {
	*enclaves.EnclaveContext
}

func InitKurtosisContext() (KurtosisCtx, error) {
	ctx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	if err != nil {
		// Kurtosis engine is probably not running. Try to start it.
		// TODO: avoid using the CLI for this
		if exec.Command("kurtosis", "engine", "start").Run() != nil {
			return KurtosisCtx{}, fmt.Errorf("failed to start Kurtosis engine: %w\n"+
				"This might be because the docker daemon is not running", err)
		}
		ctx, err = kurtosis_context.NewKurtosisContextFromLocalEngine()
	}
	return KurtosisCtx{ctx}, err
}

func (kCtx KurtosisCtx) EnclaveExists(ctx context.Context, devnetName string) bool {
	_, err := kCtx.KurtosisContext.GetEnclave(ctx, devnetName)
	return err == nil
}

func (kCtx KurtosisCtx) GetEnclaveCtx(ctx context.Context, devnetName string) (EnclaveCtx, error) {
	enclaveContext, err := kCtx.KurtosisContext.GetEnclaveContext(ctx, devnetName)
	return EnclaveCtx{enclaveContext}, err
}
