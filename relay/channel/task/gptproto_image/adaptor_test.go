package gptproto_image

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
)

func TestFetchTaskUsesGPTProtoPredictionResultEndpoint(t *testing.T) {
	var gotPath string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"id":"pred_123","status":"processing"}}`))
	}))
	defer server.Close()

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sk-test", map[string]any{"task_id": "pred_123"}, "")
	if err != nil {
		t.Fatalf("FetchTask returned error: %v", err)
	}
	_ = resp.Body.Close()

	if gotPath != "/api/v3/predictions/pred_123/result" {
		t.Fatalf("path = %q, want /api/v3/predictions/pred_123/result", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("Authorization = %q, want Bearer sk-test", gotAuth)
	}
}

func TestBearerAuthorizationDoesNotDuplicateBearerPrefix(t *testing.T) {
	if got := bearerAuthorization("Bearer sk-test"); got != "Bearer sk-test" {
		t.Fatalf("authorization = %q, want Bearer sk-test", got)
	}
}

func TestParseTaskResultMapsProcessingToInProgress(t *testing.T) {
	adaptor := &TaskAdaptor{}
	taskInfo, err := adaptor.ParseTaskResult([]byte(`{"data":{"id":"pred_123","status":"processing"}}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if taskInfo.Status != model.TaskStatusInProgress {
		t.Fatalf("status = %q, want %q", taskInfo.Status, model.TaskStatusInProgress)
	}
	if taskInfo.Progress == "" {
		t.Fatalf("progress is empty")
	}
}

func TestParseTaskResultMapsCompletedImageURL(t *testing.T) {
	adaptor := &TaskAdaptor{}
	taskInfo, err := adaptor.ParseTaskResult([]byte(`{
		"data":{
			"id":"pred_123",
			"status":"completed",
			"outputs":["https://example.com/cat.png"]
		}
	}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if taskInfo.Status != model.TaskStatusSuccess {
		t.Fatalf("status = %q, want %q", taskInfo.Status, model.TaskStatusSuccess)
	}
	if taskInfo.Url != "https://example.com/cat.png" {
		t.Fatalf("url = %q, want image URL", taskInfo.Url)
	}
	if taskInfo.Progress != "100%" {
		t.Fatalf("progress = %q, want 100%%", taskInfo.Progress)
	}
}

func TestParseTaskResultMapsFailureReason(t *testing.T) {
	adaptor := &TaskAdaptor{}
	taskInfo, err := adaptor.ParseTaskResult([]byte(`{
		"data":{
			"id":"pred_123",
			"status":"failed",
			"error":{"message":"bad prompt"}
		}
	}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if taskInfo.Status != model.TaskStatusFailure {
		t.Fatalf("status = %q, want %q", taskInfo.Status, model.TaskStatusFailure)
	}
	if taskInfo.Reason != "bad prompt" {
		t.Fatalf("reason = %q, want bad prompt", taskInfo.Reason)
	}
}

func TestParseTaskResultMapsUsageTokens(t *testing.T) {
	adaptor := &TaskAdaptor{}
	taskInfo, err := adaptor.ParseTaskResult([]byte(`{
		"data":{
			"id":"pred_123",
			"status":"completed",
			"outputs":["https://example.com/cat.png"],
			"usage":{
				"input_tokens":368,
				"output_tokens":805,
				"total_tokens":1173
			}
		}
	}`))
	if err != nil {
		t.Fatalf("ParseTaskResult returned error: %v", err)
	}
	if taskInfo.PromptTokens != 368 {
		t.Fatalf("prompt tokens = %d, want 368", taskInfo.PromptTokens)
	}
	if taskInfo.CompletionTokens != 805 {
		t.Fatalf("completion tokens = %d, want 805", taskInfo.CompletionTokens)
	}
	if taskInfo.TotalTokens != 1173 {
		t.Fatalf("total tokens = %d, want 1173", taskInfo.TotalTokens)
	}
}

func TestAdjustBillingOnCompleteUsesGPTProtoImageUsageForAdvancedTextRule(t *testing.T) {
	inputPrice := 5.0
	outputPrice := 30.0
	task := &model.Task{
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				GroupRatio:       1,
				BillingMode:      types.BillingModeAdvanced,
				AdvancedRuleType: types.AdvancedRuleTypeTextSegment,
				AdvancedRuleSnapshot: &types.AdvancedRuleSnapshot{
					RuleType:    types.AdvancedRuleTypeTextSegment,
					BillingUnit: types.AdvancedBillingUnitPerMillionTokens,
					PriceSnapshot: types.AdvancedRulePriceSnapshot{
						InputPrice:  &inputPrice,
						OutputPrice: &outputPrice,
					},
				},
			},
		},
	}
	taskInfo := &relaycommon.TaskInfo{
		PromptTokens:     368,
		CompletionTokens: 805,
		TotalTokens:      1173,
	}

	actualQuota := (&TaskAdaptor{}).AdjustBillingOnComplete(task, taskInfo)

	if actualQuota != 12995 {
		t.Fatalf("actual quota = %d, want 12995", actualQuota)
	}
	if float64(actualQuota)/common.QuotaPerUnit != 0.02599 {
		t.Fatalf("actual cost = %.6f, want 0.025990", float64(actualQuota)/common.QuotaPerUnit)
	}
}
