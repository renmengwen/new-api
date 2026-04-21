package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
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
	user, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	userUsableGroups := service.GetUserUsableGroups(user.Group)
	switch c.Query("mode") {
	case "token":
		userUsableGroups = service.ResolveUserTokenSelectableGroups(user.Group, user.GetSetting())
	case "assignable_token":
		userUsableGroups = service.ResolveAssignableTokenGroups(user)
	}

	for groupName, desc := range userUsableGroups {
		if groupName == "auto" {
			if desc == "" {
				desc = setting.GetUsableGroupDescription("auto")
			}
			usableGroups["auto"] = map[string]interface{}{
				"ratio": "鑷姩",
				"desc":  desc,
			}
			continue
		}
		if desc == "" {
			desc = setting.GetUsableGroupDescription(groupName)
		}
		usableGroups[groupName] = map[string]interface{}{
			"ratio": service.GetUserGroupRatio(user.Group, groupName),
			"desc":  desc,
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}
