package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetAdminAuditLogs(c *gin.Context) {
	operator, err := service.ResolveOperatorUser(c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if operator.Role != common.RoleRootUser && operator.GetUserType() != model.UserTypeRoot && operator.GetUserType() != model.UserTypeAdmin {
		common.ApiError(c, errors.New("permission denied"))
		return
	}
	if !requireAdminActionPermission(c, service.ResourceAuditManagement, service.ActionRead) {
		return
	}
	pageInfo := common.GetPageQuery(c)
	actionModule := c.Query("action_module")
	operatorUserID, _ := strconv.Atoi(c.Query("operator_user_id"))

	items, total, err := service.ListAdminAuditLogs(pageInfo, actionModule, operatorUserID)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}
