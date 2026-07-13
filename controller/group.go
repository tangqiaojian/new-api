package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func GetGroups(c *gin.Context) {
	groupNames := make([]string, 0)
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupNames = append(groupNames, groupName)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    groupNames,
	})
}

func GetUserGroups(c *gin.Context) {
	usableGroups := make(map[string]map[string]interface{})
	userId := c.GetInt("id")
	userCache, err := model.GetUserCache(userId)
	userGroup := ""
	if err == nil && userCache != nil {
		userGroup = userCache.Group
	}
	var userUsableGroups map[string]string
	if err == nil && userCache != nil {
		userUsableGroups = service.GetUserUsableGroupsByUser(userCache)
	} else {
		userUsableGroups = service.GetUserUsableGroups(userGroup)
	}
	userGroups := []string{userGroup}
	if err == nil && userCache != nil {
		userGroups = userCache.GetGroups()
	}
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		// UserUsableGroups contains the groups that the user can use
		if desc, ok := userUsableGroups[groupName]; ok {
			usableGroups[groupName] = map[string]interface{}{
				"ratio": bestUserGroupRatio(userGroups, groupName),
				"desc":  desc,
			}
		}
	}
	if _, ok := userUsableGroups["auto"]; ok {
		usableGroups["auto"] = map[string]interface{}{
			"ratio": "自动",
			"desc":  setting.GetUsableGroupDescription("auto"),
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}

// bestUserGroupRatio 在用户拥有的多个分组中，取该 groupName 的最优（最低）倍率。
// 倍率越低对用户越优惠，因此多分组时取最小值。
func bestUserGroupRatio(userGroups []string, groupName string) float64 {
	best := service.GetUserGroupRatio(userGroups[0], groupName)
	for _, ug := range userGroups[1:] {
		if r := service.GetUserGroupRatio(ug, groupName); r < best {
			best = r
		}
	}
	return best
}
