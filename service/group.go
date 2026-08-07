package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

func GetUserUsableGroups(userGroup string) map[string]string {
	groupsCopy := setting.GetUserUsableGroupsCopy()
	if userGroup != "" {
		specialSettings, b := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Get(userGroup)
		if b {
			// 处理特殊可用分组
			for specialGroup, desc := range specialSettings {
				if strings.HasPrefix(specialGroup, "-:") {
					// 移除分组
					groupToRemove := strings.TrimPrefix(specialGroup, "-:")
					delete(groupsCopy, groupToRemove)
				} else if strings.HasPrefix(specialGroup, "+:") {
					// 添加分组
					groupToAdd := strings.TrimPrefix(specialGroup, "+:")
					groupsCopy[groupToAdd] = desc
				} else {
					// 直接添加分组
					groupsCopy[specialGroup] = desc
				}
			}
		}
		// 如果userGroup不在UserUsableGroups中，返回UserUsableGroups + userGroup
		if _, ok := groupsCopy[userGroup]; !ok {
			groupsCopy[userGroup] = "用户分组"
		}
	}
	return groupsCopy
}

func GroupInUserUsableGroups(userGroup, groupName string) bool {
	_, ok := GetUserUsableGroups(userGroup)[groupName]
	return ok
}

// GetUserUsableGroupsByUser 从用户缓存解析其可用分组集合。
// 始终采用【严格白名单】：只能使用管理员分配的分组（Groups 多分组，或 Groups 为空时
// 回退到主 Group）及其各自「+:」规则追加的分组，不再继承全局 UserUsableGroups 中
// 未分配的分组。这样 API 密钥 / 游乐场的分组选择与管理员分配一致。
func GetUserUsableGroupsByUser(user *model.UserBase) map[string]string {
	if user == nil {
		return setting.GetUserUsableGroupsCopy()
	}
	return getUserUsableGroupsStrict(user.GetGroups())
}

// getUserUsableGroupsStrict 实现严格白名单：用户只能使用被显式分配的分组，
// 再叠加这些分组各自的「+:/-:」特殊规则。不继承全局 UserUsableGroups 中未分配的分组。
func getUserUsableGroupsStrict(userGroups []string) map[string]string {
	result := make(map[string]string)
	removals := make(map[string]bool)
	for _, userGroup := range userGroups {
		if userGroup == "" {
			continue
		}
		// 被分配的分组本身一定可用
		result[userGroup] = setting.GetUsableGroupDescription(userGroup)
		specialSettings, b := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.Get(userGroup)
		if !b {
			continue
		}
		for specialGroup, desc := range specialSettings {
			if strings.HasPrefix(specialGroup, "-:") {
				removals[strings.TrimPrefix(specialGroup, "-:")] = true
			} else if strings.HasPrefix(specialGroup, "+:") {
				result[strings.TrimPrefix(specialGroup, "+:")] = desc
			} else {
				result[specialGroup] = desc
			}
		}
	}
	// 统一应用移除：被「-:」移除的组若不在用户显式拥有的分组中，则移除
	owned := make(map[string]bool, len(userGroups))
	for _, g := range userGroups {
		owned[g] = true
	}
	for g := range removals {
		if !owned[g] {
			delete(result, g)
		}
	}
	return result
}

// GroupInUsableGroups 判断 groupName 是否在给定的可用分组集合中。
func GroupInUsableGroups(usableGroups map[string]string, groupName string) bool {
	_, ok := usableGroups[groupName]
	return ok
}

// IsUserSelectableGroup reports whether a non-auto group is both permitted by
// the user's usable-group whitelist and present in the group ratio table.
func IsUserSelectableGroup(userGroup, groupName string) bool {
	if groupName == "" || groupName == "auto" {
		return false
	}
	return GroupInUserUsableGroups(userGroup, groupName) && ratio_setting.ContainsGroupRatio(groupName)
}

// IsUserSelectableGroupByGroups is the multi-group variant of
// IsUserSelectableGroup. The whitelist is the strict whitelist computed from
// the user's assigned groups (multi-group aware) rather than the single
// primary group.
func IsUserSelectableGroupByGroups(userGroups []string, groupName string) bool {
	if groupName == "" || groupName == "auto" {
		return false
	}
	return GroupInUsableGroups(getUserUsableGroupsStrict(userGroups), groupName) && ratio_setting.ContainsGroupRatio(groupName)
}

// GetUserAutoGroup 根据用户分组获取自动分组设置
func GetUserAutoGroup(userGroup string) []string {
	autoGroups := make([]string, 0)
	seen := make(map[string]struct{})
	for _, group := range setting.GetAutoGroups() {
		if !IsUserSelectableGroup(userGroup, group) {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		autoGroups = append(autoGroups, group)
	}
	return autoGroups
}

// FilterUserTokenAutoGroups applies current permissions before the current
// per-token limit. It intentionally does not fall back to the global Auto list.
func FilterUserTokenAutoGroups(userGroup string, groups []string) []string {
	maxCount := setting.GetMaxTokenAutoGroups()
	filtered := make([]string, 0, min(len(groups), maxCount))
	seen := make(map[string]struct{})
	for _, group := range groups {
		if !IsUserSelectableGroup(userGroup, group) {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		filtered = append(filtered, group)
		if len(filtered) == maxCount {
			break
		}
	}
	return filtered
}

// FilterUserTokenAutoGroupsByGroups is the multi-group variant of
// FilterUserTokenAutoGroups. It filters the per-token Auto list against the
// strict whitelist computed from the user's assigned groups.
func FilterUserTokenAutoGroupsByGroups(userGroups []string, groups []string) []string {
	maxCount := setting.GetMaxTokenAutoGroups()
	filtered := make([]string, 0, min(len(groups), maxCount))
	seen := make(map[string]struct{})
	for _, group := range groups {
		if !IsUserSelectableGroupByGroups(userGroups, group) {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		filtered = append(filtered, group)
		if len(filtered) == maxCount {
			break
		}
	}
	return filtered
}

// GetRequestAutoGroups resolves the ordered Auto groups for the current token.
// The absence of the context value means that the token inherits the complete
// global Auto list; a present (even empty) value is an explicit token snapshot.
//
// Multi-group users (those whose assigned Groups field yields more than one
// group, or a single group different from the legacy primary Group) get a
// strict whitelist computed from their assigned groups. Single-group users on
// the legacy primary Group keep the original upstream behavior so existing
// tokens and tests continue to resolve against the global usable-group pool.
func GetRequestAutoGroups(c *gin.Context, userGroup string) []string {
	userGroups := GetUserGroupsFromCtx(c)
	useStrict := userHasExplicitMultiGroups(userGroups, userGroup)
	value, ok := common.GetContextKey(c, constant.ContextKeyTokenAutoGroups)
	if !ok {
		if useStrict {
			return GetUserAutoGroupByGroups(userGroups)
		}
		return GetUserAutoGroup(userGroup)
	}
	groups, ok := value.([]string)
	if !ok {
		return []string{}
	}
	if useStrict {
		return FilterUserTokenAutoGroupsByGroups(userGroups, groups)
	}
	return FilterUserTokenAutoGroups(userGroup, groups)
}

// userHasExplicitMultiGroups reports whether the request's user context carries
// an explicit multi-group assignment distinct from the legacy primary group.
// When true, the strict whitelist should be used; otherwise the original
// global-pool behavior is preserved for backward compatibility.
func userHasExplicitMultiGroups(userGroups []string, userGroup string) bool {
	if len(userGroups) > 1 {
		return true
	}
	if len(userGroups) == 1 && userGroup != "" && userGroups[0] != userGroup {
		return true
	}
	return false
}

// GetGroupsEnabledModels 按 groups 顺序获取各分组启用的模型并去重
func GetGroupsEnabledModels(groups []string) []string {
	seen := make(map[string]struct{})
	models := make([]string, 0)
	for _, group := range groups {
		for _, modelName := range model.GetGroupEnabledModels(group) {
			if _, ok := seen[modelName]; !ok {
				seen[modelName] = struct{}{}
				models = append(models, modelName)
			}
		}
	}
	return models
}

// GetUserGroupRatio 获取用户使用某个分组的倍率
// userGroup 用户分组
// group 需要获取倍率的分组
func GetUserGroupRatio(userGroup, group string) float64 {
	ratio, ok := ratio_setting.GetGroupGroupRatio(userGroup, group)
	if ok {
		return ratio
	}
	return ratio_setting.GetGroupRatio(group)
}

// GetUserAutoGroupByUser 为拥有多个分组的用户计算自动分组设置，
// 对每个分组分别计算其可用分组中的自动分组，再按全局顺序合并去重。
func GetUserAutoGroupByUser(user *model.UserBase) []string {
	if user == nil {
		return nil
	}
	return GetUserAutoGroupByGroups(user.GetGroups())
}

// GetUserAutoGroupByGroups 基于用户拥有的多个分组切片，计算 auto 分组（按全局顺序合并去重）。
// 注意：此函数假设调用方已确认用户被显式分配了多分组，因此采用严格白名单。
func GetUserAutoGroupByGroups(userGroups []string) []string {
	usable := getUserUsableGroupsStrict(userGroups)
	autoGroups := make([]string, 0)
	seen := make(map[string]bool)
	for _, group := range setting.GetAutoGroups() {
		if _, ok := usable[group]; ok && !seen[group] {
			seen[group] = true
			autoGroups = append(autoGroups, group)
		}
	}
	return autoGroups
}

// GetUserAutoGroupFromCtx 从请求上下文读取用户的分配分组，计算其 auto 分组。
// 与 GetRequestAutoGroups 不同，这里忽略 token 自己的 auto_groups 快照，
// 直接按用户分配的分组计算完整 auto 列表，供 distributor affinity 等场景使用。
func GetUserAutoGroupFromCtx(c *gin.Context) []string {
	return GetUserAutoGroupByGroups(GetUserGroupsFromCtx(c))
}

// GetUserUsableGroupsFromCtx 从请求上下文读取用户的分配分组，严格白名单计算可用分组。
func GetUserUsableGroupsFromCtx(c *gin.Context) map[string]string {
	return getUserUsableGroupsStrict(GetUserGroupsFromCtx(c))
}

// GetUserGroupsFromCtx 从请求上下文读取用户拥有的多个分组切片。
// 优先用多分组上下文（由 user cache WriteContext 注入），回退到单分组上下文。
func GetUserGroupsFromCtx(c *gin.Context) []string {
	if val, ok := common.GetContextKey(c, constant.ContextKeyUserGroups); ok {
		if groups, ok := val.([]string); ok && len(groups) > 0 {
			return groups
		}
	}
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	if userGroup != "" {
		return []string{userGroup}
	}
	return nil
}
