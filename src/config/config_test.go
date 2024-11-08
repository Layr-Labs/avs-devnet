package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadDefaultConfig(t *testing.T) {
	_, err := Unmarshal([]byte(DefaultConfig))
	assert.NoError(t, err, "Couldn't parse default config")
}

func TestLoadingExampleConfigs(t *testing.T) {
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
			_, err := LoadFromPath(filepath.Join(examplesDir, fileName))
			assert.NoError(t, err, "Couldn't load config from path")
		})
	}
}
