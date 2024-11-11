package cmds

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Layr-Labs/avs-devnet/src/cmds/flags"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestDefaultDevnetConfigRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, t.Name()+"_devnet.yaml")

	app := NewCliApp()
	t.Log("Creating devnet config file")
	appRun(t, app, "init", configFile)
	assert.FileExists(t, configFile, "Config file wasn't created")

	t.Log("Starting devnet from config file")
	appRun(t, app, "start", packageFlag(), configFile)

	t.Log("Stopping devnet")
	appRun(t, app, "stop", packageFlag(), configFile)
}

func appRun(t *testing.T, app *cli.App, args ...string) {
	args = append([]string{app.Name}, args...)
	err := app.Run(args)
	assert.NoError(t, err, "Failed to run '"+strings.Join(args, " ")+"'")
}

func packageFlag() string {
	return "--" + flags.KurtosisPackageFlag.Name + "=../../kurtosis_package"
}
