package cmds

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/stretchr/testify/assert"
)

func TestStartDefaultDevnet(t *testing.T) {
	opts := StartOptions{
		KurtosisPackageUrl: "../../kurtosis_package",
		DevnetName:         t.Name(),
		DevnetConfig:       config.DefaultConfig(),
	}
	ctx := context.Background()
	// Cleanup devnet after test
	t.Cleanup(func() { Stop(ctx, opts.DevnetName) })

	err := Start(ctx, opts)
	assert.NoError(t, err, "Failed to start new devnet")
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
		t.Run(fileName, func(t *testing.T) {
			examplePath := filepath.Join(examplesDir, fileName)
			devnetConfig, err := config.LoadFromPath(examplePath)
			assert.NoError(t, err, "Failed to parse example config")

			name, err := ToValidEnclaveName(t.Name())
			assert.NoError(t, err, "Failed to generate test name")

			opts := StartOptions{
				KurtosisPackageUrl: "../../kurtosis_package",
				DevnetName:         name,
				DevnetConfig:       devnetConfig,
			}
			ctx := context.Background()
			// Cleanup devnet after test
			t.Cleanup(func() { Stop(ctx, opts.DevnetName) })

			t.Log("Running test with devnet name:", opts.DevnetName)
			err = Start(ctx, opts)
			assert.NoError(t, err, "Failed to start new devnet")
		})
	}
}
