package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterSuperAdminFieldsRemovesModelMappingForNonRoot(t *testing.T) {
	logs := []*Log{{
		ChannelName: "upstream channel",
		Other: common.MapToJsonStr(map[string]interface{}{
			"request_headers":      map[string]interface{}{"x-test": "value"},
			"request_body":         `{"model":"mapped-model"}`,
			"is_model_mapped":      true,
			"upstream_model_name":  "mapped-model",
			"api_reasoning_effort": "high",
			"reasoning_effort":     "medium",
		}),
	}}

	FilterSuperAdminFields(logs, common.RoleAdminUser)

	filtered, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.NotContains(t, filtered, "request_headers")
	assert.NotContains(t, filtered, "request_body")
	assert.NotContains(t, filtered, "is_model_mapped")
	assert.NotContains(t, filtered, "upstream_model_name")
	assert.Equal(t, "high", filtered["api_reasoning_effort"])
	assert.Equal(t, "medium", filtered["reasoning_effort"])
	assert.Equal(t, "upstream channel", logs[0].ChannelName)
}

func TestFilterSuperAdminFieldsPreservesModelMappingForRoot(t *testing.T) {
	logs := []*Log{{
		Other: common.MapToJsonStr(map[string]interface{}{
			"request_headers":     map[string]interface{}{"x-test": "value"},
			"request_body":        `{"model":"mapped-model"}`,
			"is_model_mapped":     true,
			"upstream_model_name": "mapped-model",
		}),
	}}

	FilterSuperAdminFields(logs, common.RoleRootUser)

	filtered, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.Contains(t, filtered, "request_headers")
	assert.Contains(t, filtered, "request_body")
	assert.Equal(t, true, filtered["is_model_mapped"])
	assert.Equal(t, "mapped-model", filtered["upstream_model_name"])
}

func TestFormatUserLogsRemovesAdminAndModelMappingDetails(t *testing.T) {
	logs := []*Log{{
		ChannelName: "upstream channel",
		Other: common.MapToJsonStr(map[string]interface{}{
			"admin_info":           map[string]interface{}{"use_channel": []interface{}{1}},
			"audit_info":           map[string]interface{}{"operator": "root"},
			"stream_status":        map[string]interface{}{"status": "error"},
			"is_model_mapped":      true,
			"upstream_model_name":  "mapped-model",
			"api_reasoning_effort": "high",
		}),
	}}

	formatUserLogs(logs, 10, common.RoleCommonUser)

	filtered, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.Empty(t, logs[0].ChannelName)
	assert.Equal(t, 11, logs[0].Id)
	assert.NotContains(t, filtered, "admin_info")
	assert.NotContains(t, filtered, "audit_info")
	assert.NotContains(t, filtered, "stream_status")
	assert.NotContains(t, filtered, "is_model_mapped")
	assert.NotContains(t, filtered, "upstream_model_name")
	assert.Equal(t, "high", filtered["api_reasoning_effort"])
}
