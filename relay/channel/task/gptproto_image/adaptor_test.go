package gptproto_image

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
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
