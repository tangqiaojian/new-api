package ratio_setting

import (
	"github.com/QuantumNous/new-api/types"
)

// defaultModelContextLimit is intentionally empty: no model is limited until an
// admin configures a value. A value of 0 also means "no limit".
var defaultModelContextLimit = map[string]int{}

var modelContextLimitMap = types.NewRWMap[string, int]()

// GetModelContextLimitMap returns a copy of the context limit map.
func GetModelContextLimitMap() map[string]int {
	return modelContextLimitMap.ReadAll()
}

// ModelContextLimit2JSONString converts the context limit map to a JSON string.
func ModelContextLimit2JSONString() string {
	return modelContextLimitMap.MarshalJSONString()
}

// UpdateModelContextLimitByJSONString updates the context limit map from a JSON string.
func UpdateModelContextLimitByJSONString(jsonStr string) error {
	return types.LoadFromJsonStringWithCallback(modelContextLimitMap, jsonStr, InvalidateExposedDataCache)
}

// GetModelContextLimit returns the configured context limit (max prompt + max_tokens)
// for a model. The second return is false when no limit is configured for the model
// (i.e. unlimited). FormatMatchingModelName is applied so wildcard/thinking-suffix
// variants resolve the same way as the ratio maps.
func GetModelContextLimit(name string) (int, bool) {
	name = FormatMatchingModelName(name)
	limit, ok := modelContextLimitMap.Get(name)
	if !ok || limit <= 0 {
		return 0, false
	}
	return limit, true
}

// GetModelContextLimitCopy returns a copy of the context limit map.
func GetModelContextLimitCopy() map[string]int {
	return modelContextLimitMap.ReadAll()
}
