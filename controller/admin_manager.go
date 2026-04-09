package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetAdminManagers(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceAdminManagement, service.ActionRead) {
		return
	}
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	items, total, err := service.ListAdminManagers(pageInfo, keyword)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func CreateAdminManager(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceAdminManagement, service.ActionCreate) {
		return
	}

	var req service.CreateAdminManagerRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	user, err := service.CreateAdminManagerWithOperator(req, c.GetInt("id"), service.ResolveOperatorUserType(c.GetInt("id"), c.GetInt("role")), c.ClientIP())
	if err != nil {
		apiUserInputError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":           user.Id,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"user_type":    user.GetUserType(),
		"status":       user.Status,
	})
}

func GetAdminManager(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceAdminManagement, service.ActionRead) {
		return
	}

	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	data, err := service.GetAdminManagerDetail(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func UpdateAdminManager(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceAdminManagement, service.ActionUpdate) {
		return
	}

	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req service.UpdateAdminManagerRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	if err := service.UpdateAdminManagerWithOperator(userId, req, c.GetInt("id"), service.ResolveOperatorUserType(c.GetInt("id"), c.GetInt("role")), c.ClientIP()); err != nil {
		apiUserInputError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"id": userId})
}

func EnableAdminManager(c *gin.Context) {
	updateAdminManagerStatus(c, common.UserStatusEnabled)
}

func DisableAdminManager(c *gin.Context) {
	updateAdminManagerStatus(c, common.UserStatusDisabled)
}

func updateAdminManagerStatus(c *gin.Context, status int) {
	if !requireAdminActionPermission(c, service.ResourceAdminManagement, service.ActionUpdateStatus) {
		return
	}

	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := service.UpdateAdminManagerStatusWithOperator(userId, status, c.GetInt("id"), service.ResolveOperatorUserType(c.GetInt("id"), c.GetInt("role")), c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":     userId,
		"status": status,
	})
}
