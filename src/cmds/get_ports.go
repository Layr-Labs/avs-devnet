package cmds

import (
	"fmt"

	"github.com/Layr-Labs/avs-devnet/src/kurtosis"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func GetPorts(ctx *cli.Context) error {
	_, devnetName, err := parseArgs(ctx)
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
	printPorts(ports)
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

func printPorts(services map[string]ServicePorts) {
	keys := maps.Keys(services)
	slices.Sort(keys)

	for _, name := range keys {
		fmt.Println(name)
		servicePorts := services[name]
		if len(servicePorts) == 0 {
			fmt.Println("  <none>")
		}
		sortedPortNames := maps.Keys(servicePorts)
		slices.Sort(sortedPortNames)
		for _, protocol := range sortedPortNames {
			fmt.Printf("  %s: %s\n", protocol, servicePorts[protocol])
		}
		fmt.Println()
	}
}
