package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type updateUserPermissionBindingRequest struct {
	ProfileId int `json:"profile_id"`
}

func GetPermissionProfiles(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionRead) {
		return
	}
	pageInfo := common.GetPageQuery(c)
	profileType := c.Query("profile_type")

	profiles, total, err := service.ListPermissionProfiles(pageInfo, profileType)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(profiles)
	common.ApiSuccess(c, pageInfo)
}

func GetPermissionUsers(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionRead) {
		return
	}
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	userType := c.Query("user_type")

	items, total, err := service.ListPermissionUsers(pageInfo, keyword, userType)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func UpdateUserPermissionBinding(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionBindProfile) {
		return
	}
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req updateUserPermissionBindingRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	binding, err := service.UpdateUserPermissionBinding(userId, req.ProfileId, c.GetInt("id"), service.ResolveOperatorUserType(c.GetInt("id"), c.GetInt("role")), c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, binding)
}
