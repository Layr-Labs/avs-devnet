package cmds

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateInitialConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := tempDir + "/test_devnet.yaml"
	err := Init(InitOptions{configFile})
	assert.NoError(t, err, "Failed to create new config file")
}
