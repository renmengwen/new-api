package controller

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

func setupUserModelsControllerTestDB(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	common.IsMasterNode = true

	if err := setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`); err != nil {
		t.Fatalf("failed to set usable groups: %v", err)
	}

	common.SQLitePath = filepath.Join(t.TempDir(), "test.db") + "?_busy_timeout=30000"
	if err := model.InitDB(); err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := model.DB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestGetUserModelsFiltersByRequestedGroup(t *testing.T) {
	setupUserModelsControllerTestDB(t)

	user := &model.User{
		Id:       1,
		Username: "tester",
		Password: "password123",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}
	if err := model.DB.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	abilities := []model.Ability{
		{Group: "default", Model: "gpt-4o-mini", ChannelId: 1, Enabled: true},
		{Group: "vip", Model: "gpt-5.3", ChannelId: 2, Enabled: true},
	}
	if err := model.DB.Create(&abilities).Error; err != nil {
		t.Fatalf("failed to create abilities: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=default", nil)
	ctx.Set("id", 1)

	GetUserModels(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var models []string
	if err := common.Unmarshal(response.Data, &models); err != nil {
		t.Fatalf("failed to decode models response: %v", err)
	}

	if len(models) != 1 || !slices.Contains(models, "gpt-4o-mini") {
		t.Fatalf("expected only default group model, got %v", models)
	}
}

func TestGetUserModelsAllowsRequestedAllowedTokenGroup(t *testing.T) {
	setupUserModelsControllerTestDB(t)

	originalGroupRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		if err := ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatios); err != nil {
			t.Fatalf("restore group ratios: %v", err)
		}
	})
	if err := ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"beta":2}`); err != nil {
		t.Fatalf("failed to set group ratios: %v", err)
	}
	if err := setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`); err != nil {
		t.Fatalf("failed to set usable groups: %v", err)
	}

	user := &model.User{
		Id:       1,
		Username: "allowed-token-group-models",
		Password: "password123",
		Group:    "default",
		Status:   common.UserStatusEnabled,
		Setting:  `{"allowed_token_groups_enabled":true,"allowed_token_groups":["default","beta"]}`,
	}
	if err := model.DB.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	abilities := []model.Ability{
		{Group: "default", Model: "gpt-4o-mini", ChannelId: 1, Enabled: true},
		{Group: "beta", Model: "gpt-5.3", ChannelId: 2, Enabled: true},
	}
	if err := model.DB.Create(&abilities).Error; err != nil {
		t.Fatalf("failed to create abilities: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=beta", nil)
	ctx.Set("id", 1)

	GetUserModels(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var models []string
	if err := common.Unmarshal(response.Data, &models); err != nil {
		t.Fatalf("failed to decode models response: %v", err)
	}

	if len(models) != 1 || !slices.Contains(models, "gpt-5.3") {
		t.Fatalf("expected only beta group model, got %v", models)
	}
}

func TestGetUserModelsRejectsAutoWhenNotTokenSelectable(t *testing.T) {
	setupUserModelsControllerTestDB(t)

	originalAutoGroups := setting.AutoGroups2JsonString()
	t.Cleanup(func() {
		if err := setting.UpdateAutoGroupsByJsonString(originalAutoGroups); err != nil {
			t.Fatalf("restore auto groups: %v", err)
		}
	})
	if err := setting.UpdateAutoGroupsByJsonString(`["vip"]`); err != nil {
		t.Fatalf("failed to set auto groups: %v", err)
	}

	user := &model.User{
		Id:       1,
		Username: "auto-not-selectable-models",
		Password: "password123",
		Group:    "default",
		Status:   common.UserStatusEnabled,
		Setting:  `{"allowed_token_groups_enabled":true,"allowed_token_groups":["default"]}`,
	}
	if err := model.DB.Create(user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	abilities := []model.Ability{
		{Group: "default", Model: "gpt-4o-mini", ChannelId: 1, Enabled: true},
		{Group: "vip", Model: "gpt-5.3", ChannelId: 2, Enabled: true},
	}
	if err := model.DB.Create(&abilities).Error; err != nil {
		t.Fatalf("failed to create abilities: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=auto", nil)
	ctx.Set("id", 1)

	GetUserModels(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var models []string
	if err := common.Unmarshal(response.Data, &models); err != nil {
		t.Fatalf("failed to decode models response: %v", err)
	}

	if slices.Contains(models, "gpt-5.3") {
		t.Fatalf("expected auto to be ignored when not token-selectable, got %v", models)
	}
	if len(models) != 1 || !slices.Contains(models, "gpt-4o-mini") {
		t.Fatalf("expected fallback to token-selectable default models, got %v", models)
	}
}
