package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type adminAuditPageResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    common.PageInfo `json:"data"`
}

func setupAdminAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	db, err := gorm.Open(sqlite.Open("file:admin_audit?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.PermissionProfile{},
		&model.PermissionProfileItem{},
		&model.UserPermissionBinding{},
		&model.AdminAuditLog{},
	))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func TestGetAdminAuditLogsReturnsLogs(t *testing.T) {
	db := setupAdminAuditTestDB(t)
	require.NoError(t, db.Create(&model.AdminAuditLog{
		OperatorUserId:   1,
		OperatorUserType: model.UserTypeAdmin,
		ActionModule:     "quota",
		ActionType:       "adjust",
		TargetType:       "user",
		TargetId:         2,
		CreatedAtTs:      common.GetTimestamp(),
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs?p=1&page_size=10", nil)
	ctx.Set("id", 999)
	ctx.Set("role", common.RoleRootUser)

	GetAdminAuditLogs(ctx)

	var response adminAuditPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)
}

func TestGetAdminAuditLogsRequiresActionPermissionForAdmin(t *testing.T) {
	db := setupAdminAuditTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	operator := model.User{
		Username:    "audit_operator_no_grant",
		Password:    "hashed-password",
		DisplayName: "Audit Operator",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		AffCode:     "auditdeny",
	}
	require.NoError(t, db.Create(&operator).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit-logs?p=1&page_size=10", nil)
	ctx.Set("id", operator.Id)
	ctx.Set("role", operator.Role)

	GetAdminAuditLogs(ctx)

	var response adminAuditPageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
}
