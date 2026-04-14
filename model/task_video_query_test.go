package model

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestTaskGetAllUserTaskFiltersByTaskIDsAndSeedancePlatforms(t *testing.T) {
	truncateTables(t)

	userID := 1001
	seedanceVolc := constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine))
	seedanceDoubao := constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo))
	otherPlatform := constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeKling))

	insertTask(t, &Task{
		TaskID:   "task_seedance_volc",
		UserId:   userID,
		Platform: seedanceVolc,
		Status:   TaskStatusQueued,
	})
	insertTask(t, &Task{
		TaskID:   "task_seedance_doubao",
		UserId:   userID,
		Platform: seedanceDoubao,
		Status:   TaskStatusSubmitted,
	})
	insertTask(t, &Task{
		TaskID:   "task_other_platform",
		UserId:   userID,
		Platform: otherPlatform,
		Status:   TaskStatusQueued,
	})
	insertTask(t, &Task{
		TaskID:   "task_other_user",
		UserId:   userID + 1,
		Platform: seedanceVolc,
		Status:   TaskStatusQueued,
	})

	queryParams := SyncTaskQueryParams{
		Platforms: []constant.TaskPlatform{seedanceVolc, seedanceDoubao},
		TaskIDs:   []string{"task_seedance_volc", "task_seedance_doubao", "task_other_platform"},
	}

	tasks := TaskGetAllUserTask(userID, 0, 10, queryParams)
	require.Len(t, tasks, 2)
	require.Equal(t, "task_seedance_doubao", tasks[0].TaskID)
	require.Equal(t, "task_seedance_volc", tasks[1].TaskID)

	total := TaskCountAllUserTask(userID, queryParams)
	require.EqualValues(t, 2, total)
}
