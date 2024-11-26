package cmds

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/stretchr/testify/assert"
)

func startDevnet(t *testing.T, devnetConfig config.DevnetConfig) {
	name, err := ToValidEnclaveName(t.Name())
	assert.NoError(t, err, "Failed to generate test name")

	opts := StartOptions{
		KurtosisPackageUrl: "../../kurtosis_package",
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
	t.Parallel()
	err := os.Chdir("../../")
	assert.NoError(t, err, "Failed to go to repo root")

	// Clone the hello-world-avs repo
	err = exec.Command("make", "examples/hello-world-avs").Run()
	assert.NoError(t, err, "Failed to make hello-world-avs repo")

	configFile := "examples/hello_world_local.yaml"
	devnetConfig, err := config.LoadFromPath(configFile)
	assert.NoError(t, err, "Failed to parse example config")

	// Move inside the repo and start the devnet
	err = os.Chdir("examples/hello-world-avs")
	assert.NoError(t, err, "Failed to go to hello-world-avs repo")
	startDevnet(t, devnetConfig)
}
