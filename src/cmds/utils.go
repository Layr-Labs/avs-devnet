package cmds

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

var Version = "development"

// Parses the main arguments from the given context
// Returns the devnet name and the configuration file name
func parseArgs(ctx *cli.Context) (devnetName string, fileName string, err error) {
	args := ctx.Args()
	if args.Len() > 1 {
		return "", "", errors.New("expected exactly 1 argument: <config-file>")
	}
	fileName = args.First()
	if fileName == "" {
		fileName = "devnet.yaml"
		devnetName = "devnet"
	} else {
		name, err := nameFromConfigFile(fileName)
		if err != nil {
			return "", "", err
		}
		devnetName = name
	}
	return fileName, devnetName, err
}

// Checks if a file exists at the given path
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

// Extracts the devnet name from the given configuration file name
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
