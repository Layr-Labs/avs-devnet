package main

import (
	"context"
	"fmt"
	"os"

	"github.com/containerd/log"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/starlark_run_config"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
)

func main() {
	err := startEnclave()
	if err != nil {
		log.L.Error(err)
	}
}

func startEnclave() error {
	log.L.Info("Creating kurtosis context")
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	if err != nil {
		log.L.Error("Failed to create kurtosis context, error: ", err)
		fmt.Println("Ensure that the kurtosis engine is running. \nYou can check the status with `kurtosis engine status`, and start the engine with `kurtosis engine start`")
		return err
	}
	enclaveName := "devnet"
	log.L.Info("Creating kurtosis enclave")
	enclaveCtx, err := kurtosisCtx.CreateEnclave(context.Background(), enclaveName)
	if err != nil {
		log.L.Error("Failed to create enclave context, error: ", err)
		return err
	}
	scriptFile := "kurtosis_package/main.star"
	starklarkScriptBytes, err := os.ReadFile(scriptFile)
	if err != nil {
		log.L.Error("Failed to read file, error: ", err)
		return err
	}

	// run the script
	starlarkScript := string(starklarkScriptBytes)
	starlarkRunConfig := starlark_run_config.NewRunStarlarkConfig()
	log.L.Info("Running starlark script")
	starlarkRunResult, err := enclaveCtx.RunStarlarkScriptBlocking(context.Background(), starlarkScript, starlarkRunConfig)
	log.L.Info(starlarkRunResult)
	log.L.Info("Finished running starlark script")

	service := "cl-1-lighthouse-geth"
	serviceCtx, err := enclaveCtx.GetServiceContext(service)
	privatePorts := serviceCtx.GetPrivatePorts()
	log.L.Info(service+" private ports = ", privatePorts)

	kurtosisCtx.StopEnclave(context.Background(), enclaveName)
	kurtosisCtx.DestroyEnclave(context.Background(), enclaveName)

	return nil
}
