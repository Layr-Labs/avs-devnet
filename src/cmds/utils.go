package cmds

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

func parseArgs(ctx *cli.Context) (string, string, error) {
	args := ctx.Args()
	if args.Len() > 1 {
		return "", "", errors.New("expected exactly 1 argument: <config-file>")
	}
	configFileName := args.First()
	var devnetName string
	if configFileName == "" {
		configFileName = "devnet.yaml"
		devnetName = "devnet"
	} else {
		name, err := nameFromConfigFile(configFileName)
		if err != nil {
			return "", "", err
		}
		devnetName = name
	}
	return configFileName, devnetName, nil
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func nameFromConfigFile(fileName string) (string, error) {
	name := filepath.Base(fileName)
	name = strings.Split(name, ".")[0]
	name = strings.ReplaceAll(name, "_", "-")
	matches, err := regexp.MatchString("^[-A-Za-z0-9]{1,60}$", name)
	if err != nil {
		// Error in regex pattern
		panic(err)
	}
	if !matches {
		return "", errors.New("Invalid devnet name: " + name)
	}
	return name, nil
}
