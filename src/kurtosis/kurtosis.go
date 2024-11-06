package kurtosis

import (
	"context"

	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
)

type KurtosisContext struct {
	*kurtosis_context.KurtosisContext
}

func InitKurtosisContext() (*KurtosisContext, error) {
	ctx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	return &KurtosisContext{ctx}, err
}

func (kCtx KurtosisContext) EnclaveExists(ctx context.Context, devnetName string) bool {
	_, err := kCtx.KurtosisContext.GetEnclave(ctx, devnetName)
	return err == nil
}
