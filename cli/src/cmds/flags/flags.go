package flags

import "github.com/urfave/cli/v2"

// This is overwritten on release builds
var defaultKurtosisPackage string = ""

var (
	KurtosisPackageFlag = cli.StringFlag{
		Name:    "kurtosis-package",
		Usage:   "Locator for the Kurtosis package to run",
		Hidden:  true,
		EnvVars: []string{"AVS_DEVNET__KURTOSIS_PACKAGE"},
		Value:   defaultKurtosisPackage,
	}

	ConfigFilePathFlag = cli.StringFlag{
		Name:  "config-file",
		Usage: "Path to the devnet configuration file",
		Value: "devnet.yaml",
	}
)
