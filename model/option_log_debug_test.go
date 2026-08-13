package model

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionValidatesRequestDebugBodyLimit(t *testing.T) {
	truncateTables(t)
	previousValue := common.LogRequestBodyMaxBytes
	previousOptionMap := common.OptionMap
	common.OptionMap = make(map[string]string)
	require.NoError(t, DB.AutoMigrate(&Option{}))
	t.Cleanup(func() {
		common.LogRequestBodyMaxBytes = previousValue
		common.OptionMap = previousOptionMap
	})

	require.NoError(t, UpdateOption("LogRequestBodyMaxBytes", "4096"))
	assert.Equal(t, 4096, common.LogRequestBodyMaxBytes)

	for _, invalid := range []string{"not-a-number", "255", strconv.Itoa(1024*1024 + 1)} {
		err := UpdateOption("LogRequestBodyMaxBytes", invalid)
		require.Error(t, err)
		assert.Equal(t, 4096, common.LogRequestBodyMaxBytes)
		assert.Equal(t, "4096", common.OptionMap["LogRequestBodyMaxBytes"])
	}
}
