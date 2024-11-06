package kurtosis

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
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
		cli, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return KurtosisCtx{}, err
		}
		config := container.Config{
			Image: "kurtosistech/engine:latest",
		}
		cli.ContainerCreate(context.Background(), &config, nil, nil, nil, "")
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
