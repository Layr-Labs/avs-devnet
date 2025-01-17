package cmds_test

import (
	"testing"

	"github.com/Layr-Labs/avs-devnet/src/cmds"
	"github.com/stretchr/testify/assert"
)

func TestGenerateInitialConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := tempDir + "/test_devnet.yaml"
	err := cmds.Init(cmds.InitOptions{configFile})
	assert.NoError(t, err, "Failed to create new config file")
}
