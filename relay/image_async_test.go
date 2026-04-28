package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
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

	task := buildImageTask("task_public", "pred_123", []byte(`{"id":"pred_123"}`), info)

	if task.Quota != 456 {
		t.Fatalf("task quota = %d, want 456", task.Quota)
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
