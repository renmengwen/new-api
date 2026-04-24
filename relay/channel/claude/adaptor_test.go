package claude

import (
	"net/http"
	"net/http/httptest"
	"testing"

	relaychannel "github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSetupRequestHeaderDeleteHeaderRemovesAnthropicBeta(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	ctx.Request.Header.Set("anthropic-beta", "advanced-tool-use-2025-11-20")

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: "test-key",
			ParamOverride: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"mode": "delete_header",
						"path": "anthropic-beta",
					},
				},
			},
		},
	}

	_, err := relaycommon.ApplyParamOverrideWithRelayInfo([]byte(`{"messages":[{"role":"user","content":"hello"}],"max_tokens":16}`), info)
	require.NoError(t, err)

	reqHeaders := http.Header{}
	adaptor := &Adaptor{}
	require.NoError(t, adaptor.SetupRequestHeader(ctx, &reqHeaders, info))

	relaychannel.ApplyDeletedHeadersToHeader(&reqHeaders, info)
	require.Empty(t, reqHeaders.Get("anthropic-beta"))
}

func TestCommonClaudeHeadersOperationTaskBudgetBetaTracksFinalBody(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName: "claude-opus-4-7",
	}

	MarkClaudeTaskBudgetBetaFromBody(ctx, []byte(`{"output_config":{"task_budget":"standard"}}`))
	reqHeaders := http.Header{}
	CommonClaudeHeadersOperation(ctx, &reqHeaders, info)
	require.Contains(t, reqHeaders.Get("anthropic-beta"), "task-budgets-2026-03-13")

	MarkClaudeTaskBudgetBetaFromBody(ctx, []byte(`{"output_config":{"effort":"high"}}`))
	reqHeaders = http.Header{}
	CommonClaudeHeadersOperation(ctx, &reqHeaders, info)
	require.NotContains(t, reqHeaders.Get("anthropic-beta"), "task-budgets-2026-03-13")
}
