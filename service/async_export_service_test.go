package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAsyncExportServiceTestDB(t *testing.T) *gorm.DB {
	return setupAsyncExportServiceTestDBWithLogger(t, nil)
}

func setupAsyncExportServiceTestDBWithLogger(t *testing.T, gormLogger logger.Interface) *gorm.DB {
	t.Helper()

	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousExportDir := common.ExportDir
	previousUsingSQLite := common.UsingSQLite
	previousUsingMySQL := common.UsingMySQL
	previousUsingPostgreSQL := common.UsingPostgreSQL
	previousRedisEnabled := common.RedisEnabled

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	config := &gorm.Config{}
	if gormLogger != nil {
		config.Logger = gormLogger
	}
	gormDB, err := gorm.Open(sqlite.Open("file:async_export_service?mode=memory&cache=shared"), config)
	require.NoError(t, err)

	model.DB = gormDB
	model.LOG_DB = gormDB
	exportDir := t.TempDir()
	common.ExportDir = new(string)
	*common.ExportDir = exportDir

	require.NoError(t, gormDB.AutoMigrate(&model.AsyncExportJob{}))

	t.Cleanup(func() {
		model.DB = serviceTestMainDB
		model.LOG_DB = serviceTestMainLogDB
		if serviceTestMainDB == nil {
			model.DB = previousDB
			model.LOG_DB = previousLogDB
		}
		common.ExportDir = previousExportDir
		common.UsingSQLite = previousUsingSQLite
		common.UsingMySQL = previousUsingMySQL
		common.UsingPostgreSQL = previousUsingPostgreSQL
		common.RedisEnabled = previousRedisEnabled

		sqlDB, err := gormDB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return gormDB
}

type recordNotFoundTraceLogger struct {
	logger.Interface
	tracedErrors []error
}

func (l *recordNotFoundTraceLogger) LogMode(level logger.LogLevel) logger.Interface {
	l.Interface = l.Interface.LogMode(level)
	return l
}

func (l *recordNotFoundTraceLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if err != nil {
		l.tracedErrors = append(l.tracedErrors, err)
	}
	l.Interface.Trace(ctx, begin, fc, err)
}

func (l *recordNotFoundTraceLogger) sawRecordNotFound() bool {
	for _, err := range l.tracedErrors {
		if err == gorm.ErrRecordNotFound {
			return true
		}
	}
	return false
}

func TestAsyncExportCreateJobStoresQueuedMetadata(t *testing.T) {
	db := setupAsyncExportServiceTestDB(t)

	job, err := CreateAsyncExportJob("usage_logs", 42, common.RoleAdminUser, map[string]any{
		"limit": 0,
		"view":  "models",
	})
	require.NoError(t, err)
	require.Equal(t, model.AsyncExportStatusQueued, job.Status)
	require.NotZero(t, job.CreatedAtTs)

	var persisted model.AsyncExportJob
	require.NoError(t, db.First(&persisted, job.Id).Error)
	require.Equal(t, model.AsyncExportStatusQueued, persisted.Status)

	var payload map[string]any
	require.NoError(t, common.UnmarshalJsonStr(persisted.PayloadJSON, &payload))
	require.Equal(t, float64(0), payload["limit"])
	require.Equal(t, "models", payload["view"])
}

func TestAsyncExportAuthorizeJobBlocksOtherUsers(t *testing.T) {
	setupAsyncExportServiceTestDB(t)

	job, err := CreateAsyncExportJob("usage_logs", 42, common.RoleAdminUser, map[string]any{"limit": 0})
	require.NoError(t, err)

	_, err = GetAuthorizedAsyncExportJob(job.Id, 99, common.RoleCommonUser)
	require.Error(t, err)

	authorized, err := GetAuthorizedAsyncExportJob(job.Id, 42, common.RoleCommonUser)
	require.NoError(t, err)
	require.Equal(t, job.Id, authorized.Id)
}

func TestAsyncExportListJobsScopesByRequester(t *testing.T) {
	setupAsyncExportServiceTestDB(t)

	ownOld, err := CreateAsyncExportJob("usage_logs", 42, common.RoleAdminUser, map[string]any{"limit": 0})
	require.NoError(t, err)
	ownNew, err := CreateAsyncExportJob("quota_ledger", 42, common.RoleAdminUser, map[string]any{"limit": 0})
	require.NoError(t, err)
	other, err := CreateAsyncExportJob("admin_audit_logs", 99, common.RoleAdminUser, map[string]any{"limit": 0})
	require.NoError(t, err)

	items, total, err := ListAuthorizedAsyncExportJobs(&common.PageInfo{Page: 1, PageSize: 10}, 42, common.RoleCommonUser, "")
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, items, 2)
	require.Equal(t, ownNew.Id, items[0].Id)
	require.Equal(t, ownOld.Id, items[1].Id)

	rootItems, rootTotal, err := ListAuthorizedAsyncExportJobs(&common.PageInfo{Page: 1, PageSize: 10}, 1, common.RoleRootUser, "")
	require.NoError(t, err)
	require.EqualValues(t, 3, rootTotal)
	require.Len(t, rootItems, 3)
	require.Equal(t, other.Id, rootItems[0].Id)
}

func TestAsyncExportListJobsFiltersStatus(t *testing.T) {
	setupAsyncExportServiceTestDB(t)

	queued, err := CreateAsyncExportJob("usage_logs", 42, common.RoleAdminUser, map[string]any{"limit": 0})
	require.NoError(t, err)
	succeeded, err := CreateAsyncExportJob("quota_ledger", 42, common.RoleAdminUser, map[string]any{"limit": 0})
	require.NoError(t, err)
	require.NoError(t, model.DB.Model(&model.AsyncExportJob{}).Where("id = ?", succeeded.Id).Updates(map[string]any{
		"status": model.AsyncExportStatusSucceeded,
	}).Error)

	items, total, err := ListAuthorizedAsyncExportJobs(&common.PageInfo{Page: 1, PageSize: 10}, 42, common.RoleCommonUser, model.AsyncExportStatusSucceeded)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, succeeded.Id, items[0].Id)

	items, total, err = ListAuthorizedAsyncExportJobs(&common.PageInfo{Page: 1, PageSize: 10}, 42, common.RoleCommonUser, "unexpected")
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, items, 2)
	require.Equal(t, queued.Id, items[1].Id)
}

func TestAsyncExportFinalizeJobWritesArtifactMetadata(t *testing.T) {
	db := setupAsyncExportServiceTestDB(t)

	job, err := CreateAsyncExportJob("usage_logs", 42, common.RoleAdminUser, map[string]any{"limit": 0})
	require.NoError(t, err)

	filePath := filepath.Join(*common.ExportDir, "artifact.xlsx")
	require.NoError(t, os.WriteFile(filePath, []byte("xlsx-bytes"), 0o644))
	require.NoError(t, FinalizeAsyncExportJob(job.Id, "artifact.xlsx", filePath, 123))

	var persisted model.AsyncExportJob
	require.NoError(t, db.First(&persisted, job.Id).Error)
	require.Equal(t, model.AsyncExportStatusSucceeded, persisted.Status)
	require.Equal(t, "artifact.xlsx", persisted.FileName)
	require.Equal(t, filePath, persisted.FilePath)
	require.EqualValues(t, 10, persisted.FileSize)
	require.EqualValues(t, 123, persisted.RowCount)
	require.NotZero(t, persisted.CompletedAtTs)
	require.True(t, persisted.ExpiresAtTs > persisted.CompletedAtTs)
}

func TestAsyncExportCleanupExpiredArtifacts(t *testing.T) {
	db := setupAsyncExportServiceTestDB(t)

	filePath := filepath.Join(*common.ExportDir, "expired.xlsx")
	require.NoError(t, os.WriteFile(filePath, []byte("xlsx-bytes"), 0o644))

	job := model.AsyncExportJob{
		JobType:         "usage_logs",
		Status:          model.AsyncExportStatusSucceeded,
		RequesterUserId: 1,
		RequesterRole:   common.RoleAdminUser,
		FileName:        "expired.xlsx",
		FilePath:        filePath,
		FileSize:        10,
		CreatedAtTs:     common.GetTimestamp() - 100,
		CompletedAtTs:   common.GetTimestamp() - 90,
		ExpiresAtTs:     common.GetTimestamp() - 1,
	}
	require.NoError(t, db.Create(&job).Error)

	expired, err := CleanupExpiredAsyncExportJobs(common.GetTimestamp())
	require.NoError(t, err)
	require.EqualValues(t, 1, expired)

	_, statErr := os.Stat(filePath)
	require.True(t, os.IsNotExist(statErr))

	var persisted model.AsyncExportJob
	require.NoError(t, db.First(&persisted, job.Id).Error)
	require.Equal(t, model.AsyncExportStatusExpired, persisted.Status)
	require.Empty(t, persisted.FilePath)
	require.Empty(t, persisted.FileName)
	require.Zero(t, persisted.FileSize)
}

func TestClaimNextAsyncExportJobReturnsNotFoundWithoutGormQueryErrorOnEmptyQueue(t *testing.T) {
	traceLogger := &recordNotFoundTraceLogger{
		Interface: logger.Default.LogMode(logger.Info),
	}
	setupAsyncExportServiceTestDBWithLogger(t, traceLogger)

	job, err := ClaimNextAsyncExportJob()
	require.Nil(t, job)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.False(t, traceLogger.sawRecordNotFound())
}
