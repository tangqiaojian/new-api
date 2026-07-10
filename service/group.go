package service

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
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

// GetUserUsableGroupsByGroups 计算用户拥有多个分组时可用的分组集合。
// 对每个分组分别计算其可用分组，再取并集。
// 这样「+:/-:」特殊规则会叠加应用到同一个结果集合上：
// 任一分组添加的组都会保留，只有当某分组的「-:」规则且该组不在最终结果中时才会移除。
// 由于多分组语义下「移除」可能产生歧义，这里采用「先全部添加，最后统一移除」的策略，
// 即先收集所有「+:/直接添加」，再统一应用所有「-:」，避免先加后删的顺序问题。
func GetUserUsableGroupsByGroups(userGroups []string) map[string]string {
	if len(userGroups) == 0 {
		return setting.GetUserUsableGroupsCopy()
	}
	if len(userGroups) == 1 {
		return GetUserUsableGroups(userGroups[0])
	}

	result := setting.GetUserUsableGroupsCopy()
	// 收集所有需要移除的分组，最后统一移除，避免顺序依赖
	removals := make(map[string]bool)
	for _, userGroup := range userGroups {
		if userGroup == "" {
			continue
		}
		// 用户自身的分组一定可用
		if _, ok := result[userGroup]; !ok {
			result[userGroup] = "用户分组"
		}
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
	// 统一应用移除：仅当被移除的组不在用户显式拥有的分组中时才移除
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

// GetUserUsableGroupsByUser 从用户缓存解析其拥有的多个分组，返回可用分组集合。
func GetUserUsableGroupsByUser(user *model.UserBase) map[string]string {
	if user == nil {
		return setting.GetUserUsableGroupsCopy()
	}
	return GetUserUsableGroupsByGroups(user.GetGroups())
}

func GroupInUserUsableGroups(userGroup, groupName string) bool {
	_, ok := GetUserUsableGroups(userGroup)[groupName]
	return ok
}

// GroupInUsableGroups 判断 groupName 是否在给定的可用分组集合中。
func GroupInUsableGroups(usableGroups map[string]string, groupName string) bool {
	_, ok := usableGroups[groupName]
	return ok
}

// GetUserAutoGroup 根据用户分组获取自动分组设置
func GetUserAutoGroup(userGroup string) []string {
	groups := GetUserUsableGroups(userGroup)
	autoGroups := make([]string, 0)
	for _, group := range setting.GetAutoGroups() {
		if _, ok := groups[group]; ok {
			autoGroups = append(autoGroups, group)
		}
	}
	return autoGroups
}

// GetUserAutoGroupByUser 为拥有多个分组的用户计算自动分组设置，
// 对每个分组分别计算其可用分组中的自动分组，再按全局顺序合并去重。
func GetUserAutoGroupByUser(user *model.UserBase) []string {
	if user == nil {
		return nil
	}
	usable := GetUserUsableGroupsByUser(user)
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
