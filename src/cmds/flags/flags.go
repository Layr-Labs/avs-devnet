package flags

import "github.com/urfave/cli/v2"

// This is overwritten on release builds.
// TODO: move to constants.
//
//nolint:gochecknoglobals // this is a constant
var DefaultKurtosisPackage string = "github.com/Layr-Labs/avs-devnet/kurtosis_package"

//nolint:gochecknoglobals // these are constants
var (
	DevnetNameFlag = cli.StringFlag{
		Name:        "name",
		TakesFile:   true,
		Aliases:     []string{"n"},
		Usage:       "Assign a name to the devnet",
		Value:       "devnet",
		DefaultText: "devnet",
	}

	// NOTE: this flag is for internal use.
	// This flag/envvar allows us to override the Kurtosis package to local copies for development.
	// This envvar is set when running `source env.sh`.
	KurtosisPackageFlag = cli.StringFlag{
		Name:    "kurtosis-package",
		Usage:   "Locator for the Kurtosis package to run",
		Hidden:  true,
		EnvVars: []string{"AVS_DEVNET__KURTOSIS_PACKAGE"},
		Value:   DefaultKurtosisPackage,
	}
)
