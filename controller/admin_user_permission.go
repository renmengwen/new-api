package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetUserPermissionTargets(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionRead) {
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := service.ListUserPermissionTargets(pageInfo, c.Query("keyword"), c.Query("user_type"), c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetUserPermissionDetail(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionRead) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data, err := service.GetUserPermissionDetail(id, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

type updateUserPermissionTemplateRequest struct {
	ProfileId int `json:"profile_id"`
}

func UpdateUserPermissionTemplate(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionBindProfile) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req updateUserPermissionTemplateRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}
	binding, err := service.UpdateUserPermissionBinding(id, req.ProfileId, c.GetInt("id"), c.GetInt("role"), service.ResolveOperatorUserType(c.GetInt("id"), c.GetInt("role")), c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, binding)
}

func UpdateUserPermissionOverrides(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionBindProfile) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req service.UserPermissionOverrideUpdateRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}
	if err := service.UpdateUserPermissionOverrides(id, req, c.GetInt("id"), c.GetInt("role")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": id})
}
