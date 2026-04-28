package relay

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestIsGPTProtoAsyncImageRequestRequiresExplicitFalse(t *testing.T) {
	falseValue := false
	trueValue := true

	if !isGPTProtoAsyncImageRequest(&dto.ImageRequest{Model: "gpt-image-2", EnableSyncMode: &falseValue}) {
		t.Fatalf("explicit false gpt-image-2 request should be async")
	}
	if isGPTProtoAsyncImageRequest(&dto.ImageRequest{Model: "gpt-image-2", EnableSyncMode: &trueValue}) {
		t.Fatalf("explicit true gpt-image-2 request should not be async")
	}
	if isGPTProtoAsyncImageRequest(&dto.ImageRequest{Model: "gpt-image-2"}) {
		t.Fatalf("missing enable_sync_mode should not be treated as async")
	}
	if isGPTProtoAsyncImageRequest(&dto.ImageRequest{Model: "dall-e-3", EnableSyncMode: &falseValue}) {
		t.Fatalf("non GPTProto image model should not be treated as async")
	}
}

func TestShouldUseGPTProtoAsyncImageRequestRequiresSupportedChannel(t *testing.T) {
	falseValue := false
	request := &dto.ImageRequest{Model: "gpt-image-2", EnableSyncMode: &falseValue}

	if shouldUseGPTProtoAsyncImageRequest(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeOpenAI,
			ChannelBaseUrl: "https://api.openai.com",
		},
	}, request) {
		t.Fatalf("plain OpenAI-compatible channel should not use native async image route")
	}

	if !shouldUseGPTProtoAsyncImageRequest(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeOpenAI,
			ChannelBaseUrl: "https://gptproto.com",
		},
	}, request) {
		t.Fatalf("supported native async image channel should use native async image route")
	}

	if !shouldUseGPTProtoAsyncImageRequest(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeCustom,
			ChannelBaseUrl: "https://example.com/api/v3/openai/gpt-image-2/text-to-image",
		},
	}, request) {
		t.Fatalf("custom native text-to-image endpoint should use native async image route")
	}
}

func TestPrepareGPTProtoAsyncImageSubmitRouteUsesNativeTextToImagePath(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl:    "https://example.com/api/v3/openai/gpt-image-2/text-to-image",
			UpstreamModelName: "gpt-image-2",
		},
	}

	prepareGPTProtoAsyncImageSubmitRoute(info, &dto.ImageRequest{Model: "gpt-image-2"})

	if info.ChannelBaseUrl != "https://example.com" {
		t.Fatalf("base URL = %q, want https://example.com", info.ChannelBaseUrl)
	}
	if info.RequestURLPath != "/api/v3/openai/gpt-image-2/text-to-image" {
		t.Fatalf("request path = %q, want native text-to-image path", info.RequestURLPath)
	}
}

func TestPrepareGPTProtoAsyncImageSubmitRouteKeepsFullURLForCustomChannel(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeCustom,
			ChannelBaseUrl:    "https://example.com/api/v3/openai/gpt-image-2/text-to-image",
			UpstreamModelName: "gpt-image-2",
		},
	}

	prepareGPTProtoAsyncImageSubmitRoute(info, &dto.ImageRequest{Model: "gpt-image-2"})

	if info.ChannelBaseUrl != "https://example.com/api/v3/openai/gpt-image-2/text-to-image" {
		t.Fatalf("custom channel URL = %q, want full native submit URL", info.ChannelBaseUrl)
	}
	if info.RequestURLPath != "/api/v3/openai/gpt-image-2/text-to-image" {
		t.Fatalf("request path = %q, want native text-to-image path", info.RequestURLPath)
	}
}

func TestPrepareGPTProtoAsyncImageSubmitRouteInitializesMissingChannelMeta(t *testing.T) {
	info := &relaycommon.RelayInfo{}

	prepareGPTProtoAsyncImageSubmitRoute(info, &dto.ImageRequest{Model: "gpt-image-2"})

	if info.ChannelMeta == nil {
		t.Fatalf("ChannelMeta is nil")
	}
	if info.RequestURLPath != "/api/v3/openai/gpt-image-2/text-to-image" {
		t.Fatalf("request path = %q, want native text-to-image path", info.RequestURLPath)
	}
}

func TestExtractGPTProtoAsyncTaskIDFromDataID(t *testing.T) {
	taskID, err := extractGPTProtoAsyncTaskID([]byte(`{
		"data":{
			"id":"pred_123",
			"urls":{"get":"https://gptproto.com/api/v3/predictions/pred_123/result"}
		}
	}`))
	if err != nil {
		t.Fatalf("extract task id: %v", err)
	}
	if taskID != "pred_123" {
		t.Fatalf("taskID = %q, want pred_123", taskID)
	}
}

func TestExtractGPTProtoAsyncTaskIDFromDataArray(t *testing.T) {
	taskID, err := extractGPTProtoAsyncTaskID([]byte(`{
		"data":[
			{
				"id":"pred_array_123",
				"urls":{"get":"https://gptproto.com/api/v3/predictions/pred_array_123/result"}
			}
		]
	}`))
	if err != nil {
		t.Fatalf("extract task id: %v", err)
	}
	if taskID != "pred_array_123" {
		t.Fatalf("taskID = %q, want pred_array_123", taskID)
	}
}

func TestExtractGPTProtoAsyncTaskIDFromResultURL(t *testing.T) {
	taskID, err := extractGPTProtoAsyncTaskID([]byte(`{
		"data":{
			"urls":{"get":"https://gptproto.com/api/v3/predictions/pred_456/result"}
		}
	}`))
	if err != nil {
		t.Fatalf("extract task id: %v", err)
	}
	if taskID != "pred_456" {
		t.Fatalf("taskID = %q, want pred_456", taskID)
	}
}

func TestExtractGPTProtoAsyncTaskIDFromURLsArray(t *testing.T) {
	taskID, err := extractGPTProtoAsyncTaskID([]byte(`{
		"data":{
			"urls":[{"get":"https://gptproto.com/api/v3/predictions/pred_urls_123/result"}]
		}
	}`))
	if err != nil {
		t.Fatalf("extract task id: %v", err)
	}
	if taskID != "pred_urls_123" {
		t.Fatalf("taskID = %q, want pred_urls_123", taskID)
	}
}

func TestExtractGPTProtoAsyncTaskIDReportsMissingIDForSynchronousImageResult(t *testing.T) {
	_, err := extractGPTProtoAsyncTaskID([]byte(`{
		"data":[{"url":"https://example.com/generated.png"}]
	}`))
	if !errors.Is(err, errAsyncImageTaskIDMissing) {
		t.Fatalf("err = %v, want errAsyncImageTaskIDMissing", err)
	}
}

func TestHandleGPTProtoAsyncImageResponseRejectsNilResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	handled, apiErr := handleGPTProtoAsyncImageResponse(c, nil, &relaycommon.RelayInfo{}, &dto.ImageRequest{Model: "gpt-image-2"})

	if !handled {
		t.Fatalf("handled = false, want true")
	}
	if apiErr == nil {
		t.Fatalf("apiErr is nil")
	}
}

func TestHandleGPTProtoAsyncImageResponseRejectsNilRelayInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(`{"data":{"id":"pred_123"}}`)),
	}

	handled, apiErr := handleGPTProtoAsyncImageResponse(c, resp, nil, &dto.ImageRequest{Model: "gpt-image-2"})

	if !handled {
		t.Fatalf("handled = false, want true")
	}
	if apiErr == nil {
		t.Fatalf("apiErr is nil")
	}
}

func TestHandleGPTProtoAsyncImageResponseRejectsNilImageRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(`{"data":{"id":"pred_123"}}`)),
	}

	handled, apiErr := handleGPTProtoAsyncImageResponse(c, resp, &relaycommon.RelayInfo{}, nil)

	if !handled {
		t.Fatalf("handled = false, want true")
	}
	if apiErr == nil {
		t.Fatalf("apiErr is nil")
	}
}

type panicReadCloser struct{}

func (panicReadCloser) Read(_ []byte) (int, error) {
	panic("read panic")
}

func (panicReadCloser) Close() error {
	return nil
}

func TestHandleGPTProtoAsyncImageResponseRecoversInternalPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	resp := &http.Response{Body: panicReadCloser{}}

	handled, apiErr := handleGPTProtoAsyncImageResponse(c, resp, &relaycommon.RelayInfo{}, &dto.ImageRequest{Model: "gpt-image-2"})

	if !handled {
		t.Fatalf("handled = false, want true")
	}
	if apiErr == nil {
		t.Fatalf("apiErr is nil")
	}
}

func TestHandleGPTProtoAsyncImageResponseInitializesTaskRelayInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldDB := model.DB
	oldLogDB := model.LOG_DB
	oldLogConsumeEnabled := common.LogConsumeEnabled
	oldBatchUpdateEnabled := common.BatchUpdateEnabled
	oldRedisEnabled := common.RedisEnabled
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
		common.LogConsumeEnabled = oldLogConsumeEnabled
		common.BatchUpdateEnabled = oldBatchUpdateEnabled
		common.RedisEnabled = oldRedisEnabled
	})

	db, err := gorm.Open(sqlite.Open("file:image_async_task_relay_info?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&model.Task{}, &model.User{}, &model.Channel{}); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	model.DB = db
	model.LOG_DB = db
	common.LogConsumeEnabled = false
	common.BatchUpdateEnabled = false
	common.RedisEnabled = false

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(`{"data":{"id":"pred_123"}}`)),
	}
	info := &relaycommon.RelayInfo{
		UserId:          1,
		OriginModelName: "gpt-image-2",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId: 1,
		},
		PriceData: types.PriceData{
			Quota:             0,
			QuotaToPreConsume: 0,
		},
	}

	handled, apiErr := handleGPTProtoAsyncImageResponse(c, resp, info, &dto.ImageRequest{Model: "gpt-image-2"})

	if !handled {
		t.Fatalf("handled = false, want true")
	}
	if apiErr != nil {
		t.Fatalf("apiErr = %v, want nil", apiErr)
	}
	if info.TaskRelayInfo == nil {
		t.Fatalf("TaskRelayInfo is nil")
	}
	if info.Action != constant.TaskTypeImageGeneration {
		t.Fatalf("action = %q, want %q", info.Action, constant.TaskTypeImageGeneration)
	}
}

func TestBuildImageTaskUsesSettledAsyncImageQuota(t *testing.T) {
	info := &relaycommon.RelayInfo{
		FinalPreConsumedQuota: 456,
		ChannelMeta:           &relaycommon.ChannelMeta{},
		TaskRelayInfo:         &relaycommon.TaskRelayInfo{},
		PriceData: types.PriceData{
			Quota:             0,
			QuotaToPreConsume: 123,
		},
	}

	task := buildImageTask("task_public", "pred_123", []byte(`{"id":"pred_123"}`), info, nil)

	if task.Quota != 456 {
		t.Fatalf("task quota = %d, want 456", task.Quota)
	}
}

func TestBuildImageTaskStoresRequestedResponseFormat(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta:   &relaycommon.ChannelMeta{},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
	}
	request := &dto.ImageRequest{Model: "gpt-image-2", ResponseFormat: "b64_json"}

	task := buildImageTask("task_public", "pred_123", []byte(`{"id":"pred_123"}`), info, request)

	if task.Properties.ResponseFormat != "b64_json" {
		t.Fatalf("response format = %q, want b64_json", task.Properties.ResponseFormat)
	}
}

func TestTaskModel2DtoUsesLocalImageContentURL(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_public",
		Platform: constant.TaskPlatformGPTProtoImage,
		Action:   constant.TaskTypeImageGeneration,
		Status:   model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			ResultURL: "iVBORw0KGgo=",
		},
	}

	got := TaskModel2Dto(task)

	if got.ResultURL != "/v1/images/generations/task_public/content" {
		t.Fatalf("result_url = %q", got.ResultURL)
	}
}

func TestAsyncImageChargedQuotaFallsBackToPreConsumePrice(t *testing.T) {
	info := &relaycommon.RelayInfo{
		FinalPreConsumedQuota: 0,
		PriceData: types.PriceData{
			Quota:             0,
			QuotaToPreConsume: 123,
		},
	}

	if quota := asyncImageChargedQuota(info); quota != 123 {
		t.Fatalf("charged quota = %d, want 123", quota)
	}
}

func TestPrepareGPTProtoAsyncImagePriceDataForSettlementPreservesAdvancedTextLogData(t *testing.T) {
	inputPrice := 5.0
	context := &types.AdvancedPricingContextSnapshot{
		BillingUnit: types.AdvancedBillingUnitPerMillionTokens,
	}
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			Quota:             0,
			QuotaToPreConsume: 5210,
			ModelRatio:        2.5,
			CompletionRatio:   6,
			BillingMode:       types.BillingModeAdvanced,
			AdvancedRuleType:  types.AdvancedRuleTypeTextSegment,
			AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{
				RuleType: types.AdvancedRuleTypeTextSegment,
				PriceSnapshot: types.AdvancedRulePriceSnapshot{
					InputPrice: &inputPrice,
				},
			},
			AdvancedPricingContext: context,
			GroupRatioInfo:         types.GroupRatioInfo{GroupRatio: 2},
		},
	}

	chargedQuota := prepareGPTProtoAsyncImagePriceDataForSettlement(info)

	if chargedQuota != 5210 {
		t.Fatalf("charged quota = %d, want 5210", chargedQuota)
	}
	if info.PriceData.Quota != 5210 {
		t.Fatalf("price quota = %d, want 5210", info.PriceData.Quota)
	}
	if info.PriceData.BillingMode != types.BillingModeAdvanced {
		t.Fatalf("billing mode = %q, want %q", info.PriceData.BillingMode, types.BillingModeAdvanced)
	}
	if info.PriceData.AdvancedRuleType != types.AdvancedRuleTypeTextSegment {
		t.Fatalf("advanced rule type = %q, want %q", info.PriceData.AdvancedRuleType, types.AdvancedRuleTypeTextSegment)
	}
	if info.PriceData.AdvancedRuleSnapshot == nil {
		t.Fatalf("advanced rule snapshot should be preserved")
	}
	if info.PriceData.AdvancedPricingContext != context {
		t.Fatalf("advanced pricing context should be preserved")
	}
	if info.PriceData.ModelPrice != 0 {
		t.Fatalf("model price = %f, want 0", info.PriceData.ModelPrice)
	}
}
