package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const asyncExportArtifactTTLSeconds int64 = 24 * 3600
const asyncExportRunningStaleTimeoutSeconds int64 = 30 * 60

type AsyncExportExecutor func(job *model.AsyncExportJob) error

var (
	asyncExportExecutorsMu sync.RWMutex
	asyncExportExecutors   = make(map[string]AsyncExportExecutor)
)

func RegisterAsyncExportExecutor(jobType string, executor AsyncExportExecutor) {
	if jobType == "" || executor == nil {
		return
	}
	asyncExportExecutorsMu.Lock()
	defer asyncExportExecutorsMu.Unlock()
	asyncExportExecutors[jobType] = executor
}

func GetAsyncExportExecutor(jobType string) AsyncExportExecutor {
	asyncExportExecutorsMu.RLock()
	defer asyncExportExecutorsMu.RUnlock()
	return asyncExportExecutors[jobType]
}

func marshalAsyncExportPayload(payload any) (string, error) {
	data, err := common.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func CreateAsyncExportJob(jobType string, requesterUserID int, requesterRole int, payload any) (*model.AsyncExportJob, error) {
	payloadJSON, err := marshalAsyncExportPayload(payload)
	if err != nil {
		return nil, err
	}
	now := common.GetTimestamp()
	job := &model.AsyncExportJob{
		JobType:         jobType,
		Status:          model.AsyncExportStatusQueued,
		RequesterUserId: requesterUserID,
		RequesterRole:   requesterRole,
		PayloadJSON:     payloadJSON,
		CreatedAtTs:     now,
	}
	return job, model.DB.Create(job).Error
}

func DecodeAsyncExportPayload(job *model.AsyncExportJob, payload any) error {
	if job == nil {
		return errors.New("async export job is nil")
	}
	if job.PayloadJSON == "" {
		return errors.New("async export payload is empty")
	}
	return common.UnmarshalJsonStr(job.PayloadJSON, payload)
}

func GetAuthorizedAsyncExportJob(jobID int64, requesterUserID int, requesterRole int) (*model.AsyncExportJob, error) {
	var job model.AsyncExportJob
	if err := model.DB.First(&job, jobID).Error; err != nil {
		return nil, err
	}
	if requesterRole == common.RoleRootUser || job.RequesterUserId == requesterUserID {
		return &job, nil
	}
	return nil, errors.New("forbidden")
}

func BuildAsyncExportJobResponse(job *model.AsyncExportJob) dto.AsyncExportJobResponse {
	return dto.AsyncExportJobResponse{
		Id:           job.Id,
		JobType:      job.JobType,
		Status:       job.Status,
		FileName:     job.FileName,
		FileSize:     job.FileSize,
		RowCount:     job.RowCount,
		ErrorMessage: job.ErrorMessage,
		CreatedAt:    job.CreatedAtTs,
		CompletedAt:  job.CompletedAtTs,
		ExpiresAt:    job.ExpiresAtTs,
		StatusURL:    fmt.Sprintf("/api/export-jobs/%d", job.Id),
		DownloadURL:  fmt.Sprintf("/api/export-jobs/%d/file", job.Id),
	}
}

func BuildAsyncExportArtifactPath(jobID int64, fileNamePrefix string) string {
	safePrefix := fileNamePrefix
	if safePrefix == "" {
		safePrefix = "export"
	}
	fileName := fmt.Sprintf("%s_%d_%s.xlsx", safePrefix, jobID, time.Now().Format("20060102_150405"))
	baseDir := os.TempDir()
	if common.ExportDir != nil && *common.ExportDir != "" {
		baseDir = *common.ExportDir
	}
	return filepath.Join(baseDir, fileName)
}

func asyncExportLimitReached(offset int, limit int) bool {
	return limit > 0 && offset >= limit
}

func trimAsyncExportItemsToLimit[T any](items []T, offset int, limit int) []T {
	if limit <= 0 {
		return items
	}
	remaining := limit - offset
	if remaining <= 0 {
		return items[:0]
	}
	if len(items) > remaining {
		return items[:remaining]
	}
	return items
}

func isAsyncExportPageDone(rowCount int, pageSize int, offset int, limit int, total int64) bool {
	if rowCount == 0 || rowCount < pageSize {
		return true
	}
	if limit > 0 && offset+rowCount >= limit {
		return true
	}
	return total > 0 && int64(offset+rowCount) >= total
}

func ClaimNextAsyncExportJob() (*model.AsyncExportJob, error) {
	var jobs []model.AsyncExportJob
	result := model.DB.Where("status = ?", model.AsyncExportStatusQueued).Order("id asc").Limit(1).Find(&jobs)
	if result.Error != nil {
		return nil, result.Error
	}
	if len(jobs) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	job := jobs[0]
	now := common.GetTimestamp()
	result = model.DB.Model(&model.AsyncExportJob{}).
		Where("id = ? AND status = ?", job.Id, model.AsyncExportStatusQueued).
		Updates(map[string]any{
			"status":     model.AsyncExportStatusRunning,
			"started_at": now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	job.Status = model.AsyncExportStatusRunning
	job.StartedAtTs = now
	return &job, nil
}

func RequeueStaleAsyncExportJobs(now int64) (int64, error) {
	staleBefore := now - asyncExportRunningStaleTimeoutSeconds
	if staleBefore <= 0 {
		return 0, nil
	}

	result := model.DB.Model(&model.AsyncExportJob{}).
		Where("status = ? AND started_at > 0 AND started_at <= ?", model.AsyncExportStatusRunning, staleBefore).
		Updates(map[string]any{
			"status":        model.AsyncExportStatusQueued,
			"started_at":    0,
			"completed_at":  0,
			"error_message": "",
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func FinalizeAsyncExportJob(jobID int64, fileName string, filePath string, rowCount int64) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	now := common.GetTimestamp()
	return model.DB.Model(&model.AsyncExportJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":       model.AsyncExportStatusSucceeded,
		"file_name":    fileName,
		"file_path":    filePath,
		"file_size":    fileInfo.Size(),
		"row_count":    rowCount,
		"error_message": "",
		"completed_at": now,
		"expires_at":   now + asyncExportArtifactTTLSeconds,
	}).Error
}

func FailAsyncExportJob(jobID int64, message string) error {
	return model.DB.Model(&model.AsyncExportJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":        model.AsyncExportStatusFailed,
		"error_message": message,
		"completed_at":  common.GetTimestamp(),
	}).Error
}

func ExpireAsyncExportJob(job *model.AsyncExportJob) error {
	if job == nil {
		return nil
	}
	if job.FilePath != "" {
		_ = os.Remove(job.FilePath)
	}
	return model.DB.Model(&model.AsyncExportJob{}).Where("id = ?", job.Id).Updates(map[string]any{
		"status":        model.AsyncExportStatusExpired,
		"file_path":     "",
		"file_name":     "",
		"file_size":     0,
		"row_count":     0,
		"error_message": "",
		"completed_at":  common.GetTimestamp(),
	}).Error
}

func CleanupExpiredAsyncExportJobs(now int64) (int64, error) {
	var jobs []model.AsyncExportJob
	if err := model.DB.Where("status = ? AND expires_at > 0 AND expires_at <= ?", model.AsyncExportStatusSucceeded, now).Find(&jobs).Error; err != nil {
		return 0, err
	}
	var expired int64
	for i := range jobs {
		if err := ExpireAsyncExportJob(&jobs[i]); err != nil {
			return expired, err
		}
		expired++
	}
	return expired, nil
}
