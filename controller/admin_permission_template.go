package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetPermissionTemplates(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionRead) {
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := service.ListPermissionTemplates(pageInfo, c.Query("profile_type"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetPermissionTemplate(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionRead) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data, err := service.GetPermissionTemplateDetail(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func CreatePermissionTemplate(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionBindProfile) {
		return
	}
	var req service.PermissionTemplateUpsertRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}
	data, err := service.CreatePermissionTemplate(req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func UpdatePermissionTemplate(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionBindProfile) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var req service.PermissionTemplateUpsertRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}
	data, err := service.UpdatePermissionTemplate(id, req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func DeletePermissionTemplate(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourcePermissionManagement, service.ActionBindProfile) {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.DeletePermissionTemplate(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": id})
}
