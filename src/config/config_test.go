package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func forEachExample(t *testing.T, testFunc func(t *testing.T, examplePath string)) {
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
			testFunc(t, filepath.Join(examplesDir, fileName))
		})
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	_, err := Unmarshal([]byte(DefaultConfig))
	assert.NoError(t, err, "Couldn't parse default config")
}

func TestLoadingExampleConfigs(t *testing.T) {
	forEachExample(t, func(t *testing.T, examplePath string) {
		_, err := LoadFromPath(examplePath)
		assert.NoError(t, err, "Couldn't load config from path")
	})
}

func TestUnmarshalAndMarshalReturnsSameString(t *testing.T) {
	forEachExample(t, func(t *testing.T, examplePath string) {
		rawCfg, err := os.ReadFile(examplePath)
		assert.NoError(t, err, "Couldn't read config file")

		cfg, err := Unmarshal(rawCfg)
		assert.NoError(t, err, "Failed to unmarshal config")

		assert.Equal(t, rawCfg, cfg.Marshal(), "Unmarshalled and marshalled config don't match")
	})
}
