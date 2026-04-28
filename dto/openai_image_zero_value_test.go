package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/tidwall/gjson"
)

func TestImageRequestPreservesExplicitFalseEnableSyncMode(t *testing.T) {
	input := []byte(`{
		"model":"gpt-image-2",
		"prompt":"cat",
		"enable_sync_mode":false
	}`)

	var req ImageRequest
	if err := common.Unmarshal(input, &req); err != nil {
		t.Fatalf("unmarshal image request: %v", err)
	}
	if req.EnableSyncMode == nil {
		t.Fatalf("EnableSyncMode is nil, want explicit false pointer")
	}
	if *req.EnableSyncMode {
		t.Fatalf("EnableSyncMode = true, want false")
	}

	encoded, err := common.Marshal(req)
	if err != nil {
		t.Fatalf("marshal image request: %v", err)
	}
	if !gjson.GetBytes(encoded, "enable_sync_mode").Exists() {
		t.Fatalf("encoded request dropped enable_sync_mode: %s", string(encoded))
	}
	if gjson.GetBytes(encoded, "enable_sync_mode").Bool() {
		t.Fatalf("encoded enable_sync_mode = true, want false: %s", string(encoded))
	}
}
