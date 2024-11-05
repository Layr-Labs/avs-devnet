package cmds

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/avs-devnet/src/cmds/flags"
	"github.com/tidwall/gjson"
	"github.com/urfave/cli/v2"
)

func GetAddress(ctx *cli.Context) error {
	args := ctx.Args()
	configFileName := ctx.String(flags.ConfigFilePathFlag.Name)
	devnetName, err := nameFromConfigFile(configFileName)
	if err != nil {
		return cli.Exit(err, 1)
	}

	err = printAddresses(args.Slice(), devnetName)

	if err != nil {
		return cli.Exit(err, 2)
	}
	return nil
}

func printAddresses(args []string, devnetName string) error {
	failed := false
	cacheDir, err := os.MkdirTemp(os.TempDir(), ".devnet_cache")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(cacheDir) }()
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
			readFile, err := readJsonArtifact(cacheDir, devnetName, artifactName)
			if err != nil {
				fmt.Println("Error reading artifact", artifactName+":", err)
				failed = true
				readFile = ""
			}
			file = string(readFile)
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
	if strings.HasPrefix(contractName, ".") {
		// This uses the absolute path
		jsonPath = "addresses" + contractName + "|@pretty"
	} else if contractName != "" {
		// This searches for `contractName` inside the json
		// Since there are multiple results, `|0` is used to get the first one
		jsonPath = "@dig:" + contractName + "|0|@pretty"
	} else {
		// This just prints the whole json
		jsonPath = "@pretty"
	}
	res := gjson.Get(string(file), jsonPath)
	if !res.Exists() {
		return "", false
	}
	return res.String(), true
}

func readJsonArtifact(cacheDir string, devnetName string, artifactName string) (string, error) {
	err := exec.Command("sh", "-c", "cd "+cacheDir+" && kurtosis files download "+devnetName+" "+artifactName).Run()
	if err != nil {
		return "", err
	}
	artifactPath := filepath.Join(cacheDir, artifactName)
	dirEntry, err := os.ReadDir(artifactPath)
	if err != nil {
		return "", err
	}
	var readFile []byte
	for _, entry := range dirEntry {
		if entry.IsDir() {
			continue
		}
		fileName := entry.Name()
		if filepath.Ext(fileName) != ".json" {
			continue
		}
		readFile, err = os.ReadFile(filepath.Join(artifactPath, fileName))
		if err != nil {
			return "", err
		}
		break
	}
	return string(readFile), nil
}
