package cmds

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/stretchr/testify/assert"
)

var rootDir string = func() string {
	rootDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return filepath.Join(rootDir, "../..")
}()

func startDevnet(t *testing.T, devnetConfig config.DevnetConfig) {
	name, err := ToValidEnclaveName(t.Name())
	assert.NoError(t, err, "Failed to generate test name")

	opts := StartOptions{
		KurtosisPackageUrl: filepath.Join(rootDir, "kurtosis_package"),
		DevnetName:         name,
		DevnetConfig:       devnetConfig,
	}
	ctx := context.Background()

	// Ensure the devnet isn't running
	_ = Stop(ctx, opts.DevnetName)
	// Cleanup devnet after test
	t.Cleanup(func() { _ = Stop(ctx, opts.DevnetName) })

	err = Start(ctx, opts)
	assert.NoError(t, err, "Failed to start new devnet")
}

func goToDir(t *testing.T, destination string) {
	dir, err := os.Getwd()
	assert.NoError(t, err, "Failed to get cwd")

	err = os.Chdir(destination)
	assert.NoError(t, err, "Failed to go to repo root")

	t.Cleanup(func() {
		// Return to the original directory
		err = os.Chdir(dir)
		// Panic if failed, to avoid running other tests in the wrong directory
		if err != nil {
			panic(err)
		}
	})
}

func TestStartDefaultDevnet(t *testing.T) {
	t.Parallel()
	startDevnet(t, config.DefaultConfig())
}

func TestStartIncredibleSquaring(t *testing.T) {
	t.Parallel()
	examplePath := "../../examples/incredible_squaring.yaml"
	parsedConfig, err := config.LoadFromPath(examplePath)
	assert.NoError(t, err, "Failed to parse example config")
	startDevnet(t, parsedConfig)
}

func TestStartLocalHelloWorld(t *testing.T) {
	// NOTE: we don't run t.Parallel() here because we need to change the working directory
	goToDir(t, "../../")

	// Clone the hello-world-avs repo
	err := exec.Command("make", "examples/hello-world-avs").Run()
	assert.NoError(t, err, "Failed to make hello-world-avs repo")

	configFile := "examples/hello_world_local.yaml"
	devnetConfig, err := config.LoadFromPath(configFile)
	assert.NoError(t, err, "Failed to parse example config")

	// Move inside the repo and start the devnet
	goToDir(t, "examples/hello-world-avs")
	startDevnet(t, devnetConfig)
}
