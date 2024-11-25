package cmds

import (
	"fmt"

	"github.com/Layr-Labs/avs-devnet/src/kurtosis"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func GetPorts(ctx *cli.Context) error {
	devnetName, _, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}

	kurtosisCtx, err := kurtosis.InitKurtosisContext()
	if err != nil {
		return cli.Exit(err, 2)
	}
	enclaveCtx, err := kurtosisCtx.GetEnclaveCtx(ctx.Context, devnetName)
	if err != nil {
		return cli.Exit(err.Error()+"\n\nFailed to find devnet '"+devnetName+"'. Maybe it's not running?", 3)
	}
	ports, err := getServicePorts(enclaveCtx)
	if err != nil {
		return cli.Exit(err, 4)
	}
	err = printPorts(ports)
	if err != nil {
		return cli.Exit(err, 5)
	}
	return nil
}

type ServicePorts map[string]string

// Returns the
func getServicePorts(enclaveCtx kurtosis.EnclaveCtx) (map[string]ServicePorts, error) {
	servicePorts := make(map[string]ServicePorts)
	services, err := enclaveCtx.GetServices()
	if err != nil {
		return servicePorts, err
	}
	getServiceCtxsArgs := make(map[string]bool)
	for _, uuid := range services {
		getServiceCtxsArgs[string(uuid)] = true
	}
	serviceCtxs, err := enclaveCtx.GetServiceContexts(getServiceCtxsArgs)
	if err != nil {
		return servicePorts, err
	}
	for _, serviceCtx := range serviceCtxs {
		ports := make(ServicePorts)
		ipAddr := serviceCtx.GetMaybePublicIPAddress()
		for protocolName, port := range serviceCtx.GetPublicPorts() {
			ports[protocolName] = fmt.Sprintf("%s:%d", ipAddr, port.GetNumber())
		}
		name := string(serviceCtx.GetServiceName())
		servicePorts[name] = ports
	}
	return servicePorts, err
}

func printPorts(services map[string]ServicePorts) error {
	out, err := yaml.Marshal(services)
	if err != nil {
		return err
	}
	fmt.Print(string(out))
	return nil
}
