package cmds

import (
	"fmt"
	"os"

	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/urfave/cli/v2"
)

// Creates a new devnet configuration with the given context
func Init(ctx *cli.Context) error {
	_, configFileName, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	if fileExists(configFileName) {
		return cli.Exit("Config file already exists: "+configFileName, 2)
	}

	fmt.Println("Creating new devnet configuration file in", configFileName)

	err = initializeConfigFile(configFileName)
	if err != nil {
		return cli.Exit(err, 3)
	}
	return nil
}

// Creates a default devnet configuration in the given file path
func initializeConfigFile(configFileName string) error {
	file, err := os.Create(configFileName)
	if err != nil {
		return err
	}
	_, err = file.WriteString(config.DefaultConfig)
	if err != nil {
		return err
	}
	return file.Close()
}
