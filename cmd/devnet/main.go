package main

import (
	"log"
	"os"

	"github.com/Layr-Labs/avs-devnet/src/cmds"
)

func main() {
	app := cmds.NewCliApp()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
