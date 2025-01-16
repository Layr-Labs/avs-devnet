package config_test

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/avs-devnet/src/config"
	"github.com/stretchr/testify/require"
)

//go:embed default_config.yaml
var defaultConfig []byte

// TODO: this is repeated in multiple tests
func forEachExample(t *testing.T, testFunc func(t *testing.T, examplePath string)) {
	examplesDir := "../../examples"
	dir, err := os.ReadDir(examplesDir)
	require.NoError(t, err, "Couldn't read examples directory")

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
			testFunc(t, filepath.Join(examplesDir, fileName))
		})
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	_, err := config.Unmarshal(defaultConfig)
	require.NoError(t, err, "Couldn't parse default config")
}

func TestLoadingExampleConfigs(t *testing.T) {
	forEachExample(t, func(t *testing.T, examplePath string) {
		_, err := config.LoadFromPath(examplePath)
		require.NoError(t, err, "Couldn't load config from path")
	})
}

func TestUnmarshalAndMarshalReturnsSameString(t *testing.T) {
	forEachExample(t, func(t *testing.T, examplePath string) {
		rawCfg, err := os.ReadFile(examplePath)
		require.NoError(t, err, "Couldn't read config file")

		cfg, err := config.Unmarshal(rawCfg)
		require.NoError(t, err, "Failed to unmarshal config")

		require.Equal(t, rawCfg, cfg.Marshal(), "Unmarshalled and marshalled config don't match")
	})
}
