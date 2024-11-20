package cmds

import (
	"errors"
	"fmt"
	"os"

	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/urfave/cli/v2"
)

// Creates a new devnet configuration with the given context
func InitCmd(ctx *cli.Context) error {
	_, configFileName, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	cfg := InitOptions{configFileName}
	err = Init(cfg)
	if err != nil {
		return cli.Exit(err, 2)
	}
	fmt.Println("Initialized configuration file:", configFileName)
	return nil
}

type InitOptions struct {
	ConfigFileName string
}

// Creates a new devnet configuration according to the config
func Init(cfg InitOptions) error {
	if fileExists(cfg.ConfigFileName) {
		return errors.New("Config file already exists: " + cfg.ConfigFileName)
	}
	file, err := os.Create(cfg.ConfigFileName)
	if err != nil {
		return err
	}
	_, err = file.WriteString(config.DefaultConfigStr())
	if err != nil {
		return err
	}
	return file.Close()
}
