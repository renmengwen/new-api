package model

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupModelMonitorStatusTable(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&ModelMonitorStatus{}))
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ModelMonitorStatus{}).Error)
	})
}

func TestUpsertModelMonitorStatusMaintainsConsecutiveFailures(t *testing.T) {
	setupModelMonitorStatusTable(t)

	firstFailure := &ModelMonitorStatus{
		ModelName:      "gpt-4o",
		ChannelId:      7,
		ChannelName:    "primary",
		ChannelType:    1,
		Status:         ModelMonitorStatusFailed,
		ResponseTimeMs: 120,
		ErrorMessage:   "upstream error",
		TestedAt:       100,
	}
	require.NoError(t, UpsertModelMonitorStatus(firstFailure))

	stored, err := GetModelMonitorStatus("gpt-4o", 7)
	require.NoError(t, err)
	require.NotNil(t, stored)
	require.Equal(t, 1, stored.ConsecutiveFailures)
	require.NotZero(t, stored.Id)

	timeout := &ModelMonitorStatus{
		ModelName:      "gpt-4o",
		ChannelId:      7,
		ChannelName:    "fallback",
		ChannelType:    2,
		Status:         ModelMonitorStatusTimeout,
		ResponseTimeMs: 30000,
		ErrorMessage:   "timeout",
		TestedAt:       101,
	}
	require.NoError(t, UpsertModelMonitorStatus(timeout))

	stored, err = GetModelMonitorStatus("gpt-4o", 7)
	require.NoError(t, err)
	require.NotNil(t, stored)
	require.Equal(t, firstFailure.Id, stored.Id)
	require.Equal(t, 2, stored.ConsecutiveFailures)
	require.Equal(t, "fallback", stored.ChannelName)
	require.Equal(t, 2, stored.ChannelType)

	success := &ModelMonitorStatus{
		ModelName:      "gpt-4o",
		ChannelId:      7,
		ChannelName:    "fallback",
		ChannelType:    2,
		Status:         ModelMonitorStatusSuccess,
		ResponseTimeMs: 90,
		TestedAt:       102,
	}
	require.NoError(t, UpsertModelMonitorStatus(success))

	stored, err = GetModelMonitorStatus("gpt-4o", 7)
	require.NoError(t, err)
	require.NotNil(t, stored)
	require.Equal(t, 0, stored.ConsecutiveFailures)
	require.Equal(t, ModelMonitorStatusSuccess, stored.Status)
	require.Equal(t, 90, stored.ResponseTimeMs)
}

func TestListModelMonitorStatusesByModels(t *testing.T) {
	setupModelMonitorStatusTable(t)

	require.NoError(t, UpsertModelMonitorStatus(&ModelMonitorStatus{
		ModelName:   "gpt-4o",
		ChannelId:   2,
		ChannelName: "second",
		Status:      ModelMonitorStatusSuccess,
		TestedAt:    200,
	}))
	require.NoError(t, UpsertModelMonitorStatus(&ModelMonitorStatus{
		ModelName:   "gpt-4o",
		ChannelId:   1,
		ChannelName: "first",
		Status:      ModelMonitorStatusFailed,
		TestedAt:    201,
	}))
	require.NoError(t, UpsertModelMonitorStatus(&ModelMonitorStatus{
		ModelName:   "claude-3-5-sonnet",
		ChannelId:   3,
		ChannelName: "other",
		Status:      ModelMonitorStatusSkipped,
		TestedAt:    202,
	}))

	statuses, err := ListModelMonitorStatusesByModels([]string{"gpt-4o"})
	require.NoError(t, err)
	require.Len(t, statuses, 2)
	require.Equal(t, 1, statuses[0].ChannelId)
	require.Equal(t, 2, statuses[1].ChannelId)

	statuses, err = ListModelMonitorStatusesByModels(nil)
	require.NoError(t, err)
	require.Empty(t, statuses)
}

func TestUpsertModelMonitorStatusConcurrentFirstWrite(t *testing.T) {
	setupModelMonitorStatusTable(t)

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errs <- UpsertModelMonitorStatus(&ModelMonitorStatus{
				ModelName:      "concurrent-model",
				ChannelId:      11,
				ChannelName:    "concurrent-channel",
				ChannelType:    1,
				Status:         ModelMonitorStatusFailed,
				ResponseTimeMs: 100 + index,
				TestedAt:       int64(300 + index),
			})
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
	var count int64
	require.NoError(t, DB.Model(&ModelMonitorStatus{}).
		Where("model_name = ? AND channel_id = ?", "concurrent-model", 11).
		Count(&count).Error)
	require.Equal(t, int64(1), count)

	stored, err := GetModelMonitorStatus("concurrent-model", 11)
	require.NoError(t, err)
	require.NotNil(t, stored)
	require.Equal(t, workers, stored.ConsecutiveFailures)
}
