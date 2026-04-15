package relay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type seedanceTaskDeleter interface {
	DeleteTask(baseURL, key, taskID, proxy string) (*http.Response, error)
}

func buildVideoTaskFetchResponse(c *gin.Context, originTask *model.Task) (respBody []byte, taskResp *dto.TaskError) {
	path := getVideoTaskRequestPath(c)
	isOpenAIVideoAPI := strings.HasPrefix(path, "/v1/videos/")

	if isOpenAIVideoAPI {
		if shouldTryRealtimeVideoTaskFetch(originTask) {
			_ = tryRealtimeFetch(originTask, true)
		}

		adaptor := GetTaskAdaptor(originTask.Platform)
		if adaptor == nil {
			taskResp = service.TaskErrorWrapperLocal(fmt.Errorf("invalid channel id: %d", originTask.ChannelId), "invalid_channel_id", http.StatusBadRequest)
			return
		}
		if converter, ok := adaptor.(channel.OpenAIVideoConverter); ok {
			respBody, err := converter.ConvertToOpenAIVideo(originTask)
			if err != nil {
				taskResp = service.TaskErrorWrapper(err, "convert_to_openai_video_failed", http.StatusInternalServerError)
				return nil, taskResp
			}
			return respBody, nil
		}
		taskResp = service.TaskErrorWrapperLocal(fmt.Errorf("not_implemented:%s", originTask.Platform), "not_implemented", http.StatusNotImplemented)
		return
	}

	if isSeedanceVideoGenerationPath(path) && isSeedanceVideoTaskPlatform(originTask.Platform) {
		seedanceTask, err := buildSeedanceVideoTask(originTask)
		if err != nil {
			taskResp = service.TaskErrorWrapper(err, "build_seedance_task_failed", http.StatusInternalServerError)
			return
		}
		respBody, err = common.Marshal(seedanceTask)
		if err != nil {
			taskResp = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
		}
		return
	}

	respBody, err := common.Marshal(dto.TaskResponse[any]{
		Code: "success",
		Data: TaskModel2Dto(originTask),
	})
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
	}
	return
}

func buildSeedanceVideoTask(task *model.Task) (*dto.SeedanceVideoTask, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}

	seedanceTask := &dto.SeedanceVideoTask{}
	raw := bytes.TrimSpace(task.Data)
	cancelledFromSnapshot := false
	if len(raw) > 0 && !bytes.Equal(raw, []byte("null")) {
		if err := common.Unmarshal(raw, seedanceTask); err != nil {
			return nil, err
		}
		cancelledFromSnapshot = strings.EqualFold(strings.TrimSpace(seedanceTask.Status), "cancelled")
	}

	seedanceTask.ID = task.TaskID
	seedanceTask.Status = mapTaskStatusToSeedance(task, cancelledFromSnapshot)

	if task.Properties.OriginModelName != "" {
		seedanceTask.Model = task.Properties.OriginModelName
	} else if task.Properties.UpstreamModelName != "" {
		seedanceTask.Model = task.Properties.UpstreamModelName
	}

	if task.CreatedAt != 0 {
		seedanceTask.CreatedAt = task.CreatedAt
	} else if seedanceTask.CreatedAt == 0 {
		seedanceTask.CreatedAt = task.SubmitTime
	}

	if task.UpdatedAt != 0 {
		seedanceTask.UpdatedAt = task.UpdatedAt
	} else if task.FinishTime != 0 {
		seedanceTask.UpdatedAt = task.FinishTime
	} else if task.StartTime != 0 {
		seedanceTask.UpdatedAt = task.StartTime
	} else if seedanceTask.UpdatedAt == 0 {
		seedanceTask.UpdatedAt = seedanceTask.CreatedAt
	}

	if resultURL := task.GetResultURL(); resultURL != "" {
		if seedanceTask.Content == nil {
			seedanceTask.Content = &dto.SeedanceVideoTaskContent{}
		}
		seedanceTask.Content.VideoURL = resultURL
	}
	if seedanceTask.Content != nil && seedanceTask.Content.VideoURL == "" {
		seedanceTask.Content = nil
	}

	return seedanceTask, nil
}

func videoFetchListRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	userID := c.GetInt("id")
	pageNum := parseSeedanceVideoPageParam(c.Query("page_num"), 1)
	pageSize := parseSeedanceVideoPageParam(c.Query("page_size"), 10)
	rawStatus := c.Query("filter.status")

	queryParams := model.SyncTaskQueryParams{
		Platforms: seedanceVideoTaskPlatforms(),
		TaskIDs:   parseSeedanceVideoTaskIDs(c),
		Statuses:  parseSeedanceVideoTaskStatuses(rawStatus),
	}
	if isSeedanceCancelledStatus(rawStatus) {
		queryParams.FailReasonContains = "cancelled"
	}

	startIdx := (pageNum - 1) * pageSize
	tasks := model.TaskGetAllUserTask(userID, startIdx, pageSize, queryParams)
	total := model.TaskCountAllUserTask(userID, queryParams)

	items := make([]*dto.SeedanceVideoTask, 0, len(tasks))
	for _, task := range tasks {
		seedanceTask, err := buildSeedanceVideoTask(task)
		if err != nil {
			return nil, service.TaskErrorWrapper(err, "build_seedance_task_failed", http.StatusInternalServerError)
		}
		items = append(items, seedanceTask)
	}

	var err error
	respBody, err = common.Marshal(dto.SeedanceVideoTaskListResponse{
		Total: total,
		Items: items,
	})
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
	}
	return
}

func videoDeleteRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskID := c.Param("task_id")
	if taskID == "" {
		taskID = c.GetString("task_id")
	}
	userID := c.GetInt("id")

	originTask, exist, err := model.GetByTaskId(userID, taskID)
	if err != nil {
		return nil, service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
	}
	if !exist {
		return nil, service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
	}
	if !isSeedanceVideoTaskPlatform(originTask.Platform) {
		return nil, service.TaskErrorWrapperLocal(errors.New("unsupported_task_platform"), "unsupported_task_platform", http.StatusBadRequest)
	}

	adaptor := GetTaskAdaptor(originTask.Platform)
	deleter, ok := adaptor.(seedanceTaskDeleter)
	if !ok {
		return nil, service.TaskErrorWrapperLocal(errors.New("task_delete_not_supported"), "task_delete_not_supported", http.StatusNotImplemented)
	}

	channelModel, err := model.GetChannelById(originTask.ChannelId, true)
	if err != nil {
		return nil, service.TaskErrorWrapper(err, "get_channel_failed", http.StatusInternalServerError)
	}

	upstreamTaskID := originTask.GetUpstreamTaskID()
	if strings.TrimSpace(upstreamTaskID) == "" {
		return nil, service.TaskErrorWrapperLocal(errors.New("upstream_task_id_not_found"), "upstream_task_id_not_found", http.StatusBadRequest)
	}

	resp, err := deleter.DeleteTask(channelModel.GetBaseURL(), channelModel.Key, upstreamTaskID, channelModel.GetSetting().Proxy)
	if err != nil {
		return nil, service.TaskErrorWrapper(err, "delete_task_failed", http.StatusInternalServerError)
	}
	if resp == nil {
		return nil, service.TaskErrorWrapperLocal(errors.New("empty_upstream_response"), "empty_upstream_response", http.StatusBadGateway)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		responseBody, _ := io.ReadAll(resp.Body)
		return nil, service.TaskErrorWrapper(fmt.Errorf("%s", string(responseBody)), "fail_to_delete_task", resp.StatusCode)
	}

	if taskResp = applySeedanceTaskDeleteLocally(c, originTask); taskResp != nil {
		return nil, taskResp
	}

	respBody, err = common.Marshal(dto.SeedanceVideoTaskDeleteResponse{})
	if err != nil {
		return nil, service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
	}
	return respBody, nil
}

func getVideoTaskRequestPath(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if c.Request.URL != nil && c.Request.URL.Path != "" {
		return c.Request.URL.Path
	}
	return c.Request.RequestURI
}

func isSeedanceVideoGenerationPath(path string) bool {
	return strings.HasPrefix(path, "/v1/video/generations")
}

func isSeedanceVideoTaskPlatform(platform constant.TaskPlatform) bool {
	return platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)) ||
		platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo))
}

func shouldTryRealtimeVideoTaskFetch(task *model.Task) bool {
	if task == nil {
		return false
	}
	return task.Platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVertexAi)) ||
		task.Platform == constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeGemini))
}

func mapTaskStatusToSeedance(task *model.Task, cancelledFromSnapshot bool) string {
	if isCancelledSeedanceTask(task, cancelledFromSnapshot) {
		return "cancelled"
	}

	switch task.Status {
	case model.TaskStatusSubmitted, model.TaskStatusQueued:
		return "pending"
	case model.TaskStatusInProgress:
		return "processing"
	case model.TaskStatusSuccess:
		return "succeeded"
	case model.TaskStatusFailure:
		return "failed"
	default:
		return "processing"
	}
}

func isCancelledSeedanceTask(task *model.Task, cancelledFromSnapshot bool) bool {
	if task == nil {
		return false
	}
	if cancelledFromSnapshot {
		return true
	}
	return task.Status == model.TaskStatusFailure &&
		strings.Contains(strings.ToLower(strings.TrimSpace(task.FailReason)), "cancelled")
}

func isSeedanceCancelledStatus(raw string) bool {
	return strings.EqualFold(strings.TrimSpace(raw), "cancelled")
}

func parseSeedanceVideoPageParam(raw string, defaultValue int) int {
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultValue
	}
	return value
}

func parseSeedanceVideoTaskIDs(c *gin.Context) []string {
	values := c.QueryArray("filter.task_ids")
	taskIDs := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			taskID := strings.TrimSpace(item)
			if taskID == "" {
				continue
			}
			if _, ok := seen[taskID]; ok {
				continue
			}
			seen[taskID] = struct{}{}
			taskIDs = append(taskIDs, taskID)
		}
	}
	return taskIDs
}

func parseSeedanceVideoTaskStatuses(raw string) []model.TaskStatus {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return nil
	case "pending":
		return []model.TaskStatus{model.TaskStatusSubmitted, model.TaskStatusQueued}
	case "processing":
		return []model.TaskStatus{model.TaskStatusInProgress}
	case "succeeded":
		return []model.TaskStatus{model.TaskStatusSuccess}
	case "failed":
		return []model.TaskStatus{model.TaskStatusFailure}
	case "cancelled":
		return []model.TaskStatus{model.TaskStatusFailure}
	default:
		return []model.TaskStatus{model.TaskStatus(strings.ToUpper(strings.TrimSpace(raw)))}
	}
}

func seedanceVideoTaskPlatforms() []constant.TaskPlatform {
	return []constant.TaskPlatform{
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVolcEngine)),
		constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)),
	}
}

func applySeedanceTaskDeleteLocally(c *gin.Context, task *model.Task) *dto.TaskError {
	if task == nil {
		return service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
	}

	if isTerminalTaskStatus(task.Status) {
		if err := model.DB.Delete(task).Error; err != nil {
			return service.TaskErrorWrapper(err, "delete_task_failed", http.StatusInternalServerError)
		}
		return nil
	}

	now := time.Now().Unix()
	oldStatus := task.Status
	task.Status = model.TaskStatusFailure
	task.Progress = "100%"
	task.FinishTime = now
	task.UpdatedAt = now
	task.FailReason = "cancelled by user"

	won, err := task.UpdateWithStatus(oldStatus)
	if err != nil {
		return service.TaskErrorWrapper(err, "update_task_failed", http.StatusInternalServerError)
	}
	if !won {
		refreshedTask, exist, getErr := model.GetByTaskId(task.UserId, task.TaskID)
		if getErr != nil {
			return service.TaskErrorWrapper(getErr, "get_task_failed", http.StatusInternalServerError)
		}
		if !exist {
			return nil
		}
		if isTerminalTaskStatus(refreshedTask.Status) {
			if err := model.DB.Delete(refreshedTask).Error; err != nil {
				return service.TaskErrorWrapper(err, "delete_task_failed", http.StatusInternalServerError)
			}
			return nil
		}
		return service.TaskErrorWrapperLocal(errors.New("task_status_conflict"), "task_status_conflict", http.StatusConflict)
	}

	if task.Quota != 0 {
		refundCtx := context.Background()
		if c != nil && c.Request != nil {
			refundCtx = c.Request.Context()
		}
		service.RefundTaskQuota(refundCtx, task, task.FailReason)
	}

	return nil
}

func isTerminalTaskStatus(status model.TaskStatus) bool {
	return status == model.TaskStatusSuccess || status == model.TaskStatusFailure
}
