package cmds

import (
	"context"
	"testing"

	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/stretchr/testify/assert"
)

func TestStartDefaultDevnet(t *testing.T) {
	cfg := StartOptions{
		KurtosisPackageUrl: "../../kurtosis_package",
		DevnetName:         t.Name(),
		DevnetConfig:       config.DefaultConfig(),
	}
	ctx := context.Background()
	// Cleanup devnet after test
	t.Cleanup(func() { Stop(ctx, cfg.DevnetName) })

	err := Start(ctx, cfg)
	assert.NoError(t, err, "Failed to start new devnet")
}
