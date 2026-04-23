package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type asyncExportAPIResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Data    map[string]any           `json:"data"`
}

func setupAsyncExportControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	gormDB, err := gorm.Open(sqlite.Open("file:async_export_controller?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = gormDB
	model.LOG_DB = gormDB
	exportDir := t.TempDir()
	common.ExportDir = new(string)
	*common.ExportDir = exportDir

	require.NoError(t, gormDB.AutoMigrate(&model.AsyncExportJob{}))

	t.Cleanup(func() {
		sqlDB, err := gormDB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return gormDB
}

func newAsyncExportTestContext(method string, target string, userID int, role int, jobID int64) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(method, target, nil)
	ctx.Request = req
	ctx.Set("id", userID)
	ctx.Set("role", role)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.FormatInt(jobID, 10)}}
	return ctx, recorder
}

func TestAsyncExportGetJobStatusReturnsJobMetadata(t *testing.T) {
	db := setupAsyncExportControllerTestDB(t)

	job := model.AsyncExportJob{
		JobType:         "usage_logs",
		Status:          model.AsyncExportStatusQueued,
		RequesterUserId: 7,
		RequesterRole:   common.RoleAdminUser,
		CreatedAtTs:     common.GetTimestamp(),
	}
	require.NoError(t, db.Create(&job).Error)

	ctx, recorder := newAsyncExportTestContext(http.MethodGet, "/api/export-jobs/1", 7, common.RoleAdminUser, job.Id)
	GetAsyncExportJob(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response asyncExportAPIResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.EqualValues(t, job.Id, response.Data["id"])
	require.Equal(t, job.JobType, response.Data["job_type"])
	require.Equal(t, job.Status, response.Data["status"])
}

func TestAsyncExportDownloadRejectsNonSucceededJob(t *testing.T) {
	db := setupAsyncExportControllerTestDB(t)

	job := model.AsyncExportJob{
		JobType:         "usage_logs",
		Status:          model.AsyncExportStatusQueued,
		RequesterUserId: 7,
		RequesterRole:   common.RoleAdminUser,
		CreatedAtTs:     common.GetTimestamp(),
	}
	require.NoError(t, db.Create(&job).Error)

	ctx, recorder := newAsyncExportTestContext(http.MethodGet, "/api/export-jobs/1/file", 7, common.RoleAdminUser, job.Id)
	DownloadAsyncExportJobFile(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response asyncExportAPIResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)
	require.Contains(t, response.Message, "not ready")
}

func TestAsyncExportDownloadStreamsSucceededArtifact(t *testing.T) {
	db := setupAsyncExportControllerTestDB(t)

	filePath := filepath.Join(*common.ExportDir, "ready.xlsx")
	require.NoError(t, os.WriteFile(filePath, []byte("xlsx-bytes"), 0o644))

	job := model.AsyncExportJob{
		JobType:         "usage_logs",
		Status:          model.AsyncExportStatusSucceeded,
		RequesterUserId: 7,
		RequesterRole:   common.RoleAdminUser,
		FileName:        "ready.xlsx",
		FilePath:        filePath,
		FileSize:        10,
		CreatedAtTs:     common.GetTimestamp(),
		CompletedAtTs:   common.GetTimestamp(),
		ExpiresAtTs:     common.GetTimestamp() + 3600,
	}
	require.NoError(t, db.Create(&job).Error)

	ctx, recorder := newAsyncExportTestContext(http.MethodGet, "/api/export-jobs/1/file", 7, common.RoleAdminUser, job.Id)
	DownloadAsyncExportJobFile(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "ready.xlsx")
	require.Equal(t, "xlsx-bytes", recorder.Body.String())
}
