package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetAgents(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceAgentManagement, service.ActionRead) {
		return
	}
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")

	items, total, err := service.ListAgents(pageInfo, keyword)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func CreateAgent(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceAgentManagement, service.ActionCreate) {
		return
	}
	var req service.CreateAgentRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiError(c, errors.New("invalid request body"))
		return
	}

	user, err := service.CreateAgentWithOperator(req, c.GetInt("id"), service.ResolveOperatorUserType(c.GetInt("id"), c.GetInt("role")), c.ClientIP())
	if err != nil {
		common.ApiError(c, err)
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

func GetAgent(c *gin.Context) {
	if !requireAdminActionPermission(c, service.ResourceAgentManagement, service.ActionRead) {
		return
	}
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	data, err := service.GetAgentDetail(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func EnableAgent(c *gin.Context) {
	updateAgentStatus(c, common.UserStatusEnabled)
}

func DisableAgent(c *gin.Context) {
	updateAgentStatus(c, common.UserStatusDisabled)
}

func updateAgentStatus(c *gin.Context, status int) {
	if !requireAdminActionPermission(c, service.ResourceAgentManagement, service.ActionUpdateStatus) {
		return
	}
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := service.UpdateAgentStatusWithOperator(userId, status, c.GetInt("id"), service.ResolveOperatorUserType(c.GetInt("id"), c.GetInt("role")), c.ClientIP()); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"id":     userId,
		"status": status,
	})
}
