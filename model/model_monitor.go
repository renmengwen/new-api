package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ModelMonitorStatusSuccess = "success"
	ModelMonitorStatusFailed  = "failed"
	ModelMonitorStatusTimeout = "timeout"
	ModelMonitorStatusSkipped = "skipped"
)

type ModelMonitorStatus struct {
	Id                  int    `json:"id" gorm:"primaryKey"`
	ModelName           string `json:"model_name" gorm:"type:varchar(255);not null;uniqueIndex:uk_model_monitor_target,priority:1;index"`
	ChannelId           int    `json:"channel_id" gorm:"not null;uniqueIndex:uk_model_monitor_target,priority:2;index"`
	ChannelName         string `json:"channel_name" gorm:"type:varchar(255);default:''"`
	ChannelType         int    `json:"channel_type" gorm:"default:0"`
	Status              string `json:"status" gorm:"type:varchar(32);not null;default:'skipped';index"`
	ResponseTimeMs      int    `json:"response_time_ms" gorm:"default:0"`
	ErrorMessage        string `json:"error_message" gorm:"type:text"`
	TestedAt            int64  `json:"tested_at" gorm:"bigint;index"`
	ConsecutiveFailures int    `json:"consecutive_failures" gorm:"default:0"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt           int64  `json:"updated_at" gorm:"bigint"`
}

func (s *ModelMonitorStatus) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	if s.CreatedAt == 0 {
		s.CreatedAt = now
	}
	if s.UpdatedAt == 0 {
		s.UpdatedAt = now
	}
	if s.TestedAt == 0 {
		s.TestedAt = now
	}
	if s.Status == "" {
		s.Status = ModelMonitorStatusSkipped
	}
	return nil
}

func (s *ModelMonitorStatus) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = common.GetTimestamp()
	if s.TestedAt == 0 {
		s.TestedAt = s.UpdatedAt
	}
	if s.Status == "" {
		s.Status = ModelMonitorStatusSkipped
	}
	return nil
}

func UpsertModelMonitorStatus(status *ModelMonitorStatus) error {
	if status == nil {
		return errors.New("model monitor status is nil")
	}
	if status.Status == "" {
		status.Status = ModelMonitorStatusSkipped
	}

	err := upsertModelMonitorStatusOnce(status)
	var createConflictErr *modelMonitorStatusCreateConflictError
	if errors.As(err, &createConflictErr) {
		retryErr := updateModelMonitorStatusAfterCreateConflict(status)
		if retryErr == nil {
			return nil
		}
		if errors.Is(retryErr, gorm.ErrRecordNotFound) {
			return createConflictErr.err
		}
		return retryErr
	}
	return err
}

type modelMonitorStatusCreateConflictError struct {
	err error
}

func (e *modelMonitorStatusCreateConflictError) Error() string {
	return e.err.Error()
}

func (e *modelMonitorStatusCreateConflictError) Unwrap() error {
	return e.err
}

func upsertModelMonitorStatusOnce(status *ModelMonitorStatus) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var existing ModelMonitorStatus
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("model_name = ? AND channel_id = ?", status.ModelName, status.ChannelId).
			First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				status.ConsecutiveFailures = nextModelMonitorFailureCount(0, status.Status)
				if createErr := tx.Create(status).Error; createErr == nil {
					return nil
				} else {
					return &modelMonitorStatusCreateConflictError{err: createErr}
				}
			}
			return err
		}

		return updateModelMonitorStatus(tx, status, existing)
	})
}

func updateModelMonitorStatusAfterCreateConflict(status *ModelMonitorStatus) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var existing ModelMonitorStatus
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("model_name = ? AND channel_id = ?", status.ModelName, status.ChannelId).
			First(&existing).Error
		if err != nil {
			return err
		}
		return updateModelMonitorStatus(tx, status, existing)
	})
}

func updateModelMonitorStatus(tx *gorm.DB, status *ModelMonitorStatus, existing ModelMonitorStatus) error {
	status.Id = existing.Id
	status.CreatedAt = existing.CreatedAt
	status.ConsecutiveFailures = nextModelMonitorFailureCount(existing.ConsecutiveFailures, status.Status)

	updates := map[string]interface{}{
		"channel_name":         status.ChannelName,
		"channel_type":         status.ChannelType,
		"status":               status.Status,
		"response_time_ms":     status.ResponseTimeMs,
		"error_message":        status.ErrorMessage,
		"tested_at":            status.TestedAt,
		"consecutive_failures": status.ConsecutiveFailures,
		"updated_at":           common.GetTimestamp(),
	}
	if status.TestedAt == 0 {
		updates["tested_at"] = updates["updated_at"]
		status.TestedAt = updates["updated_at"].(int64)
	}
	status.UpdatedAt = updates["updated_at"].(int64)

	return tx.Model(&ModelMonitorStatus{}).Where("id = ?", existing.Id).Updates(updates).Error
}

func ListModelMonitorStatusesByModels(modelNames []string) ([]*ModelMonitorStatus, error) {
	if len(modelNames) == 0 {
		return []*ModelMonitorStatus{}, nil
	}
	var statuses []*ModelMonitorStatus
	err := DB.Where("model_name IN ?", modelNames).
		Order("model_name asc, channel_id asc").
		Find(&statuses).Error
	return statuses, err
}

func GetModelMonitorStatus(modelName string, channelId int) (*ModelMonitorStatus, error) {
	var status ModelMonitorStatus
	err := DB.Where("model_name = ? AND channel_id = ?", modelName, channelId).First(&status).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &status, nil
}

func nextModelMonitorFailureCount(current int, status string) int {
	if status == ModelMonitorStatusFailed || status == ModelMonitorStatusTimeout {
		return current + 1
	}
	return 0
}
