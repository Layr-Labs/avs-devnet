package cmds

import (
	"fmt"
	"os"

	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/urfave/cli/v2"
)

func Init(ctx *cli.Context) error {
	configFileName, _, err := parseArgs(ctx)
	if err != nil {
		return cli.Exit(err, 1)
	}
	if fileExists(configFileName) {
		return cli.Exit("Config file already exists: "+configFileName, 2)
	}

	fmt.Println("Creating new devnet configuration file in", configFileName)

	file, err := os.Create(configFileName)
	if err != nil {
		return cli.Exit(err, 3)
	}
	_, err = file.WriteString(config.DefaultConfig)
	if err != nil {
		return cli.Exit(err, 4)
	}
	return file.Close()
}
