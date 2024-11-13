package flags

import "github.com/urfave/cli/v2"

// This is overwritten on release builds
var defaultKurtosisPackage string = ""

var (
	ConfigFilePathFlag = cli.StringFlag{
		Name:  "config-file",
		Usage: "Path to the devnet configuration file",
		Value: "devnet.yaml",
	}

	// NOTE: this flag is for internal use.
	// This flag/envvar allows us to override the Kurtosis package to local copies for development.
	// This envvar is set when running `source env.sh`.
	KurtosisPackageFlag = cli.StringFlag{
		Name:    "kurtosis-package",
		Usage:   "Locator for the Kurtosis package to run",
		Hidden:  true,
		EnvVars: []string{"AVS_DEVNET__KURTOSIS_PACKAGE"},
		Value:   defaultKurtosisPackage,
	}
)
