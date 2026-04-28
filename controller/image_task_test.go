package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupImageTaskTestDB(t *testing.T) {
	t.Helper()

	oldDB := model.DB
	oldLogDB := model.LOG_DB
	db, err := gorm.Open(sqlite.Open("file:image_task_controller?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.Task{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db

	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestGetImageGenerationTaskReturnsDocumentedPublicShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupImageTaskTestDB(t)

	task := &model.Task{
		TaskID:     "task_public",
		UserId:     7,
		Platform:   constant.TaskPlatformGPTProtoImage,
		Action:     constant.TaskTypeImageGeneration,
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		SubmitTime: 1777356946,
		Properties: model.Properties{
			OriginModelName: "gpt-image-2",
		},
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://oss.gptproto.com/ai-draw/result.png",
		},
		Data: []byte(`{"data":{"urls":{"get":"https://gptproto.com/api/v3/predictions/upstream/result"},"outputs":["https://oss.gptproto.com/ai-draw/result.png"]}}`),
	}
	if err := model.DB.Create(task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "https://api.example.test/v1/images/generations/task_public", nil)
	ctx.Params = gin.Params{{Key: "task_id", Value: "task_public"}}
	ctx.Set("id", 7)

	GetImageGenerationTask(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if strings.Contains(body, "gptproto") {
		t.Fatalf("response leaked upstream content: %s", body)
	}

	var payload map[string]any
	if err := common.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := payload["data"]; ok {
		t.Fatalf("response should not expose raw task data: %s", body)
	}
	if payload["task_id"] != "task_public" {
		t.Fatalf("task_id = %v, want task_public", payload["task_id"])
	}
	if payload["platform"] != "image" {
		t.Fatalf("platform = %v, want image", payload["platform"])
	}
	if payload["action"] != constant.TaskTypeImageGeneration {
		t.Fatalf("action = %v, want %s", payload["action"], constant.TaskTypeImageGeneration)
	}
	if payload["status"] != string(model.TaskStatusSuccess) {
		t.Fatalf("status = %v, want %s", payload["status"], model.TaskStatusSuccess)
	}
	if payload["result_url"] != "https://api.example.test/v1/images/generations/task_public/content" {
		t.Fatalf("result_url = %v", payload["result_url"])
	}
}
