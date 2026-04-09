package controller

import (
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
