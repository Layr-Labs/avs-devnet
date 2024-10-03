package main

import (
	"context"
	"fmt"

	"github.com/containerd/log"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
)

func main() {
	fmt.Println("Hello world")
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	if err != nil {
		log.L.Error("Failed to create kurtosis context, error = ", err)
		fmt.Println("Ensure that the kurtosis engine is running. \nYou can check the status with `kurtosis engine status`, and start the engine with `kurtosis engine start`")
		return
	}
	enclaveName := "devnet"
	enclaveCtx, err := kurtosisCtx.CreateEnclave(context.Background(), enclaveName)
	if err != nil {
		log.L.Error("Failed to create enclave context, error =", err)
		return
	}
	name := enclaveCtx.GetEnclaveName()
	fmt.Println("enclave name = ", name)
}
