package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Layr-Labs/avs-devnet/src/cmds"
)

func main() {
	app := cmds.NewCliApp()

	fmt.Println(os.Args)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
