package cmds

import (
	"context"
	"os"
	"path/filepath"
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

func TestStartExampleDevnets(t *testing.T) {
	examplesDir := "../../examples"
	dir, err := os.ReadDir(examplesDir)
	assert.NoError(t, err, "Couldn't read examples directory")

	for _, file := range dir {
		if file.IsDir() {
			continue
		}
		fileName := file.Name()
		ext := filepath.Ext(fileName)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		examplePath := filepath.Join(examplesDir, fileName)
		parsedConfig, err := config.LoadFromPath(examplePath)
		assert.NoError(t, err, "Failed to parse example config")

		t.Run(fileName, func(t *testing.T) {
			// Don't reference variables outside function
			devnetConfig := parsedConfig
			t.Parallel()
			startDevnet(t, devnetConfig)
		})
	}
}
