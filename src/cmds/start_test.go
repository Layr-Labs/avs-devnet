package cmds_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/avs-devnet/src/cmds"
	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals // these are constants used for tests
var (
	rootDir string = func() string {
		cwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		return filepath.Join(cwd, "../..")
	}()

	examplesDir string = filepath.Join(rootDir, "examples")
)

func startDevnet(t *testing.T, devnetConfig config.DevnetConfig, workingDir string) {
	name, err := cmds.ToValidEnclaveName(t.Name())
	require.NoError(t, err, "Failed to generate test name")

	opts := cmds.StartOptions{
		KurtosisPackageUrl: filepath.Join(rootDir, "kurtosis_package"),
		DevnetName:         name,
		WorkingDir:         workingDir,
		DevnetConfig:       devnetConfig,
	}
	ctx := context.Background()

	// Ensure the devnet isn't running
	_ = cmds.Stop(ctx, opts.DevnetName)
	// Cleanup devnet after test
	t.Cleanup(func() { _ = cmds.Stop(ctx, opts.DevnetName) })

	err = cmds.Start(ctx, opts)
	require.NoError(t, err, "Failed to start new devnet")
}

func TestStartDefaultDevnet(t *testing.T) {
	t.Parallel()
	startDevnet(t, config.DefaultConfig(), examplesDir)
}

func TestStartIncredibleSquaring(t *testing.T) {
	t.Parallel()
	examplePath := filepath.Join(examplesDir, "incredible_squaring.yaml")
	parsedConfig, err := config.LoadFromPath(examplePath)
	require.NoError(t, err, "Failed to parse example config")
	startDevnet(t, parsedConfig, examplesDir)
}

func TestStartLocalHelloWorld(t *testing.T) {
	t.Parallel()
	// Clone the hello-world-avs repo
	err := exec.Command("sh", "-s", "cd ../../ && make examples/hello-world-avs").Run()
	require.NoError(t, err, "Failed to make hello-world-avs repo")

	configFile := filepath.Join(examplesDir, "hello_world_local.yaml")
	devnetConfig, err := config.LoadFromPath(configFile)
	require.NoError(t, err, "Failed to parse example config")

	// Start the devnet
	helloWorldRepo := filepath.Join(examplesDir, "hello-world-avs")
	startDevnet(t, devnetConfig, helloWorldRepo)
}
