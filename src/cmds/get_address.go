package cmds

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Layr-Labs/avs-devnet/src/cmds/flags"
	"github.com/Layr-Labs/avs-devnet/src/kurtosis"
	"github.com/kurtosis-tech/kurtosis/api/golang/core/lib/services"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v2"
)

func GetAddress(ctx *cli.Context) error {
	args := ctx.Args()
	devnetName := flags.DevnetNameFlag.Get(ctx)

	kurtosisCtx, err := kurtosis.InitKurtosisContext()
	if err != nil {
		return cli.Exit(err, 1)
	}
	enclaveCtx, err := kurtosisCtx.GetEnclaveCtx(ctx.Context, devnetName)
	if err != nil {
		return cli.Exit(err.Error()+"\n\nFailed to find devnet '"+devnetName+"'. Maybe it's not running?", 1)
	}

	err = printAddresses(ctx, args.Slice(), enclaveCtx)

	if err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func printAddresses(ctx *cli.Context, args []string, enclaveCtx kurtosis.EnclaveCtx) error {
	failed := false
	cached := make(map[string]string)

	for _, arg := range args {
		// contract name is like "artifact-name:contract-name"
		path := strings.Split(arg, ":")
		// TODO: assume a length of 1 means it's just the contract name
		if len(path) > 2 || len(path) == 1 {
			fmt.Println("Invalid contract name: " + arg)
			failed = true
			continue
		}
		artifactName := path[0]
		contractName := path[1]
		file, ok := cached[artifactName]
		if !ok {
			readFile, err := readJsonArtifact(ctx, enclaveCtx, artifactName)
			if err != nil {
				fmt.Println("Error reading artifact", artifactName+":", err)
				failed = true
				readFile = ""
			}
			file = readFile
			cached[artifactName] = file
		}
		output, ok := readArtifact(file, contractName)
		if !ok {
			fmt.Println("Error getting", arg)
			failed = true
			continue
		}
		fmt.Println(strings.TrimSpace(output))
	}
	if failed {
		return errors.New("failed to get some addresses")
	}
	return nil
}

func readArtifact(file string, contractName string) (string, bool) {
	var jsonPath string
	switch {
	case strings.HasPrefix(contractName, "."):
		// This uses the absolute path
		jsonPath = "addresses" + contractName + "|@pretty"
	case contractName != "":
		// This searches for `contractName` inside the json
		// Since there are multiple results, `|0` is used to get the first one
		jsonPath = "@dig:" + contractName + "|0|@pretty"
	default:
		// This just prints the whole json
		jsonPath = "@pretty"
	}
	res := gjson.Get(file, jsonPath)
	if !res.Exists() {
		return "", false
	}
	return res.String(), true
}

func readJsonArtifact(ctx *cli.Context, enclaveCtx kurtosis.EnclaveCtx, artifactName string) (string, error) {
	artifactInfo, err := enclaveCtx.InspectFilesArtifact(ctx.Context, services.FileArtifactName(artifactName))
	if err != nil {
		return "", err
	}
	for _, file := range artifactInfo.GetFileDescriptions() {
		if strings.HasSuffix(file.GetPath(), ".json") {
			return file.GetTextPreview(), nil
		}
	}
	return "", errors.New("No json file found in artifact " + artifactName)
}
