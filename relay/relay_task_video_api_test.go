package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	taskdoubao "github.com/QuantumNous/new-api/relay/channel/task/doubao"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBuildSeedanceVideoTaskFromPolledSnapshot(t *testing.T) {
	raw, err := common.Marshal(dto.SeedanceVideoTask{
		ID:        "upstream_task_x",
		Model:     "doubao-seedance-1-5-pro-251215",
		Status:    "succeeded",
		CreatedAt: 1,
		UpdatedAt: 2,
		Content: &dto.SeedanceVideoTaskContent{
			VideoURL: "https://upstream.example/video.mp4",
		},
	})
	require.NoError(t, err)

	task := &model.Task{
		TaskID:    "task_public_x",
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
		Status:    model.TaskStatusSuccess,
		CreatedAt: 1710000000,
		UpdatedAt: 1710000300,
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-1-5-pro-251215",
		},
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://gateway.example/video.mp4",
		},
		Data: raw,
	}

	got, err := buildSeedanceVideoTask(task)
	require.NoError(t, err)
	require.Equal(t, task.TaskID, got.ID)
	require.Equal(t, "succeeded", got.Status)
	require.Equal(t, task.Properties.OriginModelName, got.Model)
	require.Equal(t, task.CreatedAt, got.CreatedAt)
	require.Equal(t, task.UpdatedAt, got.UpdatedAt)
	require.NotNil(t, got.Content)
	require.Equal(t, task.PrivateData.ResultURL, got.Content.VideoURL)
}

func TestBuildSeedanceVideoTaskFromSubmitSnapshot(t *testing.T) {
	raw, err := common.Marshal(map[string]any{
		"id": "upstream_task_x",
	})
	require.NoError(t, err)

	task := &model.Task{
		TaskID:    "task_public_x",
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)),
		Status:    model.TaskStatusQueued,
		CreatedAt: 1710000000,
		UpdatedAt: 1710000001,
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-1-0-lite-i2v",
		},
		Data: raw,
	}

	got, err := buildSeedanceVideoTask(task)
	require.NoError(t, err)
	require.Equal(t, task.TaskID, got.ID)
	require.Equal(t, "pending", got.Status)
	require.Equal(t, task.Properties.OriginModelName, got.Model)
	require.Equal(t, task.CreatedAt, got.CreatedAt)
	require.Equal(t, task.UpdatedAt, got.UpdatedAt)
	require.Nil(t, got.Content)
}

func TestDoubaoDoResponseReturnsSeedanceCreateResponseForOfficialAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"id":"upstream_task_x"}`)),
	}

	adaptor := &taskdoubao.TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		OriginModelName: "doubao-seedance-1-5-pro-251215",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public_x",
		},
	}

	taskID, taskData, taskErr := adaptor.DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	require.Equal(t, "upstream_task_x", taskID)
	require.JSONEq(t, `{"id":"task_public_x"}`, recorder.Body.String())
	require.JSONEq(t, `{"id":"upstream_task_x"}`, string(taskData))
}

func TestDoubaoDoResponseKeepsOpenAIVideoResponseForLegacyVideosAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"id":"upstream_task_x"}`)),
	}

	adaptor := &taskdoubao.TaskAdaptor{}
	info := &relaycommon.RelayInfo{
		OriginModelName: "doubao-seedance-1-5-pro-251215",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public_x",
		},
	}

	taskID, _, taskErr := adaptor.DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	require.Equal(t, "upstream_task_x", taskID)

	var got dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &got))
	require.Equal(t, "task_public_x", got.ID)
	require.Equal(t, "task_public_x", got.TaskID)
	require.Equal(t, "video", got.Object)
}

func TestBuildVideoTaskFetchResponseUsesSeedanceFormatOnlyForSeedancePlatforms(t *testing.T) {
	gin.SetMode(gin.TestMode)

	seedanceTask := &model.Task{
		TaskID:    "task_seedance_x",
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
		Status:    model.TaskStatusSuccess,
		CreatedAt: 1710000000,
		UpdatedAt: 1710000300,
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-1-5-pro-251215",
		},
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://gateway.example/video.mp4",
		},
	}
	seedanceTask.Data = mustMarshalSeedanceTask(t, dto.SeedanceVideoTask{
		ID:        "upstream_task_x",
		Model:     "doubao-seedance-1-5-pro-251215",
		Status:    "succeeded",
		CreatedAt: 1,
		UpdatedAt: 2,
		Content: &dto.SeedanceVideoTaskContent{
			VideoURL: "https://upstream.example/video.mp4",
		},
	})

	seedanceContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	seedanceContext.Request = httptest.NewRequest(http.MethodGet, "/v1/video/generations/task_seedance_x", nil)
	seedanceBody, taskErr := buildVideoTaskFetchResponse(seedanceContext, seedanceTask)
	require.Nil(t, taskErr)

	var officialTask dto.SeedanceVideoTask
	require.NoError(t, common.Unmarshal(seedanceBody, &officialTask))
	require.Equal(t, seedanceTask.TaskID, officialTask.ID)
	require.Equal(t, "succeeded", officialTask.Status)

	otherTask := &model.Task{
		TaskID:    "task_other_x",
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeJimeng)),
		Status:    model.TaskStatusSuccess,
		CreatedAt: 1710000000,
		UpdatedAt: 1710000300,
	}

	otherContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	otherContext.Request = httptest.NewRequest(http.MethodGet, "/v1/video/generations/task_other_x", nil)
	otherBody, taskErr := buildVideoTaskFetchResponse(otherContext, otherTask)
	require.Nil(t, taskErr)

	var genericResp dto.TaskResponse[map[string]any]
	require.NoError(t, common.Unmarshal(otherBody, &genericResp))
	require.Equal(t, dto.TaskSuccessCode, genericResp.Code)
	require.Equal(t, otherTask.TaskID, genericResp.Data["task_id"])
}

func TestBuildVideoTaskFetchResponseKeepsOpenAIVideoForLegacyVideosAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	task := &model.Task{
		TaskID:    "task_seedance_x",
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
		Status:    model.TaskStatusSuccess,
		CreatedAt: 1710000000,
		UpdatedAt: 1710000300,
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-1-5-pro-251215",
		},
	}
	task.Data = mustMarshalSeedanceTask(t, dto.SeedanceVideoTask{
		ID:     "upstream_task_x",
		Model:  "doubao-seedance-1-5-pro-251215",
		Status: "succeeded",
		Content: &dto.SeedanceVideoTaskContent{
			VideoURL: "https://upstream.example/video.mp4",
		},
	})

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/videos/task_seedance_x", nil)

	respBody, taskErr := buildVideoTaskFetchResponse(c, task)
	require.Nil(t, taskErr)

	var got dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(respBody, &got))
	require.Equal(t, task.TaskID, got.ID)
	require.Equal(t, task.TaskID, got.TaskID)
	require.Equal(t, dto.VideoStatusCompleted, got.Status)
}

func TestRelayTaskFetchVideoListReturnsSeedancePage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initRelayTaskVideoListTestDB(t)

	insertRelayVideoTask(t, &model.Task{
		TaskID:    "task_seedance_1",
		UserId:    7,
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
		Status:    model.TaskStatusQueued,
		CreatedAt: 1710000001,
		UpdatedAt: 1710000101,
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-1-5-pro-251215",
		},
	})
	insertRelayVideoTask(t, &model.Task{
		TaskID:    "task_seedance_2",
		UserId:    7,
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)),
		Status:    model.TaskStatusSubmitted,
		CreatedAt: 1710000002,
		UpdatedAt: 1710000102,
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-1-0-lite-i2v",
		},
	})
	insertRelayVideoTask(t, &model.Task{
		TaskID:    "task_seedance_done",
		UserId:    7,
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
		Status:    model.TaskStatusSuccess,
		CreatedAt: 1710000003,
		UpdatedAt: 1710000103,
		Properties: model.Properties{
			OriginModelName: "doubao-seedance-2-0-pro",
		},
	})
	insertRelayVideoTask(t, &model.Task{
		TaskID:    "task_kling_ignored",
		UserId:    7,
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeKling)),
		Status:    model.TaskStatusQueued,
		CreatedAt: 1710000004,
		UpdatedAt: 1710000104,
	})
	insertRelayVideoTask(t, &model.Task{
		TaskID:    "task_other_user",
		UserId:    8,
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
		Status:    model.TaskStatusQueued,
		CreatedAt: 1710000005,
		UpdatedAt: 1710000105,
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/v1/video/generations?page_num=2&page_size=1&filter.status=pending&filter.task_ids=task_seedance_1,task_seedance_2&filter.task_ids=task_seedance_done",
		nil,
	)
	c.Set("id", 7)

	taskErr := RelayTaskFetch(c, relayconstant.RelayModeVideoFetchList)
	require.Nil(t, taskErr)

	var got dto.SeedanceVideoTaskListResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &got))
	require.EqualValues(t, 2, got.Total)
	require.Len(t, got.Items, 1)
	require.Equal(t, "task_seedance_1", got.Items[0].ID)
	require.Equal(t, "pending", got.Items[0].Status)
}

func TestRelayTaskFetchVideoDeleteMarksRunningSeedanceTaskAsCancelled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initRelayTaskVideoListTestDB(t)
	service.InitHttpClient()

	deleteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalls++
		require.Equal(t, http.MethodDelete, r.Method)
		require.Equal(t, "/api/v3/contents/generations/tasks/upstream_running_x", r.URL.Path)
		require.Equal(t, "Bearer delete-key", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	insertRelayVideoChannel(t, &model.Channel{
		Id:      301,
		Type:    constant.ChannelTypeVolcEngine,
		Key:     "delete-key",
		BaseURL: &server.URL,
	})
	insertRelayVideoTask(t, &model.Task{
		TaskID:    "task_running_x",
		UserId:    7,
		ChannelId: 301,
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
		Status:    model.TaskStatusInProgress,
		Progress:  "50%",
		Quota:     0,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream_running_x",
		},
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodDelete, "/v1/video/generations/task_running_x", nil)
	c.Params = gin.Params{{Key: "task_id", Value: "task_running_x"}}
	c.Set("id", 7)

	taskErr := RelayTaskFetch(c, relayconstant.RelayModeVideoDelete)
	require.Nil(t, taskErr)
	require.JSONEq(t, `{}`, recorder.Body.String())
	require.Equal(t, 1, deleteCalls)

	reloaded, exist, err := model.GetByTaskId(7, "task_running_x")
	require.NoError(t, err)
	require.True(t, exist)
	require.Equal(t, model.TaskStatus(model.TaskStatusFailure), reloaded.Status)
	require.Equal(t, "100%", reloaded.Progress)
	require.Equal(t, "cancelled by user", reloaded.FailReason)
	require.NotZero(t, reloaded.FinishTime)
}

func TestRelayTaskFetchVideoDeleteRemovesTerminalSeedanceTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initRelayTaskVideoListTestDB(t)
	service.InitHttpClient()

	deleteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalls++
		require.Equal(t, http.MethodDelete, r.Method)
		require.Equal(t, "/api/v3/contents/generations/tasks/upstream_success_x", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	insertRelayVideoChannel(t, &model.Channel{
		Id:      302,
		Type:    constant.ChannelTypeDoubaoVideo,
		Key:     "delete-key",
		BaseURL: &server.URL,
	})
	insertRelayVideoTask(t, &model.Task{
		TaskID:    "task_success_x",
		UserId:    7,
		ChannelId: 302,
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)),
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		Quota:     0,
		PrivateData: model.TaskPrivateData{
			UpstreamTaskID: "upstream_success_x",
		},
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodDelete, "/v1/video/generations/task_success_x", nil)
	c.Params = gin.Params{{Key: "task_id", Value: "task_success_x"}}
	c.Set("id", 7)

	taskErr := RelayTaskFetch(c, relayconstant.RelayModeVideoDelete)
	require.Nil(t, taskErr)
	require.JSONEq(t, `{}`, recorder.Body.String())
	require.Equal(t, 1, deleteCalls)

	_, exist, err := model.GetByTaskId(7, "task_success_x")
	require.NoError(t, err)
	require.False(t, exist)
}

func TestRelayTaskFetchVideoDeleteRejectsNonSeedancePlatforms(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initRelayTaskVideoListTestDB(t)
	service.InitHttpClient()

	deleteCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	insertRelayVideoChannel(t, &model.Channel{
		Id:      303,
		Type:    constant.ChannelTypeJimeng,
		Key:     "delete-key",
		BaseURL: &server.URL,
	})
	insertRelayVideoTask(t, &model.Task{
		TaskID:    "task_other_platform_x",
		UserId:    7,
		ChannelId: 303,
		Platform:  constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeJimeng)),
		Status:    model.TaskStatusInProgress,
		Progress:  "50%",
		Quota:     0,
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodDelete, "/v1/video/generations/task_other_platform_x", nil)
	c.Params = gin.Params{{Key: "task_id", Value: "task_other_platform_x"}}
	c.Set("id", 7)

	taskErr := RelayTaskFetch(c, relayconstant.RelayModeVideoDelete)
	require.NotNil(t, taskErr)
	require.Equal(t, 0, deleteCalls)

	reloaded, exist, err := model.GetByTaskId(7, "task_other_platform_x")
	require.NoError(t, err)
	require.True(t, exist)
	require.Equal(t, model.TaskStatus(model.TaskStatusInProgress), reloaded.Status)
}

func mustMarshalSeedanceTask(t *testing.T, task dto.SeedanceVideoTask) []byte {
	t.Helper()

	raw, err := common.Marshal(task)
	require.NoError(t, err)
	return raw
}

func initRelayTaskVideoListTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	model.DB = db
	model.LOG_DB = db
	common.UsingSQLite = true

	require.NoError(t, db.AutoMigrate(&model.Task{}, &model.Channel{}))
}

func insertRelayVideoTask(t *testing.T, task *model.Task) {
	t.Helper()
	require.NoError(t, model.DB.Create(task).Error)
}

func insertRelayVideoChannel(t *testing.T, channel *model.Channel) {
	t.Helper()
	require.NoError(t, model.DB.Create(channel).Error)
}
