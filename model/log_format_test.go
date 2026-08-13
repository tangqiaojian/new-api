package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/require"
)

// TestFormatUserLogsStripsQuotaSaturation verifies the admin-only quota
// saturation marker (nested under other.admin_info) is removed for non-admin
// log views, since formatUserLogs strips the whole admin_info object.
func TestFormatUserLogsStripsQuotaSaturation(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"model_price": 0.004,
		"admin_info": map[string]interface{}{
			"quota_saturation": map[string]interface{}{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
			},
		},
	})
	logs := []*Log{{Other: other}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (and nested quota_saturation) must be stripped for non-admin views")
	// Non-admin billing fields remain visible.
	require.Contains(t, parsed, "model_price")
}

func TestFilterSuperAdminFieldsPreservesDebugDataOnlyForRoot(t *testing.T) {
	debugOther := common.MapToJsonStr(map[string]interface{}{
		"request_headers": map[string]interface{}{"X-Trace": "safe"},
		"request_body":    "payload",
		"model_price":     0.004,
	})

	adminLogs := []*Log{{Other: debugOther}}
	FilterSuperAdminFields(adminLogs, common.RoleAdminUser)
	adminOther, err := common.StrToMap(adminLogs[0].Other)
	require.NoError(t, err)
	require.NotContains(t, adminOther, "request_headers")
	require.NotContains(t, adminOther, "request_body")
	require.Contains(t, adminOther, "model_price")

	rootLogs := []*Log{{Other: debugOther}}
	FilterSuperAdminFields(rootLogs, common.RoleRootUser)
	rootOther, err := common.StrToMap(rootLogs[0].Other)
	require.NoError(t, err)
	require.Contains(t, rootOther, "request_headers")
	require.Equal(t, "payload", rootOther["request_body"])
}
