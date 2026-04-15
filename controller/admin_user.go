package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetAdminUsers(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceUserManagement, service.ActionRead) {
		return
	}
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	items, total, err := service.ListAdminUsers(pageInfo, keyword, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetAdminUser(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceUserManagement, service.ActionRead) {
		return
	}
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	data, err := service.GetAdminUserDetail(userId, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func CreateAdminUser(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceUserManagement, service.ActionCreate) {
		return
	}

	var req service.CreateAdminUserRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	user, err := service.CreateAdminUserWithOperator(req, c.GetInt("id"), c.GetInt("role"), c.ClientIP())
	if err != nil {
		apiUserInputError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":              user.Id,
		"username":        user.Username,
		"display_name":    user.DisplayName,
		"user_type":       user.GetUserType(),
		"status":          user.Status,
		"parent_agent_id": user.ParentAgentId,
	})
}

func UpdateAdminUser(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceUserManagement, service.ActionUpdate) {
		return
	}

	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req service.UpdateAdminUserRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	if err := service.UpdateAdminUserWithOperator(userId, req, c.GetInt("id"), c.GetInt("role"), c.ClientIP()); err != nil {
		apiUserInputError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"id": userId})
}

func DeleteAdminUser(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceUserManagement, service.ActionDelete) {
		return
	}

	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := service.DeleteAdminUserWithOperator(userId, c.GetInt("id"), c.GetInt("role"), c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{"id": userId})
}

func EnableAdminUser(c *gin.Context) {
	updateAdminUserStatus(c, common.UserStatusEnabled)
}

func DisableAdminUser(c *gin.Context) {
	updateAdminUserStatus(c, common.UserStatusDisabled)
}

func updateAdminUserStatus(c *gin.Context, status int) {
	if !requireAdminActionPermission(c, service.ResourceUserManagement, service.ActionUpdateStatus) {
		return
	}
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := service.UpdateAdminUserStatus(userId, status, c.GetInt("id"), c.GetInt("role"), c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"id":     userId,
		"status": status,
	})
}
