package cmds

import (
	"context"
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
	// Cleanup devnet after test
	t.Cleanup(func() { _ = Stop(ctx, opts.DevnetName) })

	err = Start(ctx, opts)
	assert.NoError(t, err, "Failed to start new devnet")
}

func TestStartDefaultDevnet(t *testing.T) {
	startDevnet(t, config.DefaultConfig())
}
