package cmds

import (
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/urfave/cli/v2"
)

// Parses the configuration file path from the positional args.
// Fails if more than one positional arg is provided.
func parseConfigFileName(ctx *cli.Context) (string, error) {
	args := ctx.Args()
	if args.Len() > 1 {
		return "", errors.New("expected none or 1 argument: [<file-name>]")
	}
	fileName := args.First()
	// TODO: check file exists and support yml extension
	if fileName == "" {
		return "devnet.yaml", nil
	}
	return fileName, nil
}

// Checks if a file exists at the given path.
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func ToValidEnclaveName(name string) (string, error) {
	for _, r := range []string{"_", "/", "."} {
		name = strings.ReplaceAll(name, r, "-")
	}
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
