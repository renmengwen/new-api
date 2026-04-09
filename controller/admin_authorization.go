package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func requireAdminActionPermission(c *gin.Context, resourceKey string, actionKey string) bool {
	if err := service.RequirePermissionAction(c.GetInt("id"), c.GetInt("role"), resourceKey, actionKey); err != nil {
		common.ApiError(c, err)
		return false
	}
	return true
}
