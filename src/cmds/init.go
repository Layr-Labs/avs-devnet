package cmds

import (
	"errors"
	"fmt"
	"os"

	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/urfave/cli/v2"
)

// Creates a new devnet configuration with the given context.
func InitCmd(ctx *cli.Context) error {
	configFileName, err := parseConfigFileName(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	opts := InitOptions{configFileName}
	err = Init(opts)
	if err != nil {
		return cli.Exit(err, 1)
	}
	fmt.Println("Initialized configuration file:", configFileName)
	return nil
}

type InitOptions struct {
	ConfigFileName string
}

// Creates a new devnet configuration according to the config.
func Init(opts InitOptions) error {
	if fileExists(opts.ConfigFileName) {
		return errors.New("Config file already exists: " + opts.ConfigFileName)
	}
	file, err := os.Create(opts.ConfigFileName)
	if err != nil {
		return err
	}
	_, err = file.WriteString(config.DefaultConfigStr())
	if err != nil {
		return err
	}
	return file.Close()
}
