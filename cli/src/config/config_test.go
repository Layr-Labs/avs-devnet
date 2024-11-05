package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadingExampleConfigs(t *testing.T) {
	dir, err := os.ReadDir("../../../examples")
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
			_, err := LoadFromPath("../../../examples/" + fileName)
			assert.NoError(t, err, "Couldn't load config from path")
		})
	}
}
