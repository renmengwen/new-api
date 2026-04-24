package claude

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func requireOutputConfigMap(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()

	require.NotEmpty(t, raw)
	var outputConfig map[string]any
	require.NoError(t, common.Unmarshal(raw, &outputConfig))
	return outputConfig
}

func requireOpus47AdaptiveThinking(t *testing.T, thinking *dto.Thinking) {
	t.Helper()

	require.NotNil(t, thinking)
	require.Equal(t, "adaptive", thinking.Type)
	require.NotNil(t, thinking.Display)
	require.Equal(t, "summarized", *thinking.Display)
	require.Nil(t, thinking.BudgetTokens)
}

func TestRequestOpenAI2ClaudeMessageOpus47HighSuffixUsesAdaptiveThinking(t *testing.T) {
	maxTokens := uint(4096)
	temperature := 0.7
	topP := 0.9
	topK := 20

	request, err := RequestOpenAI2ClaudeMessage(nil, dto.GeneralOpenAIRequest{
		Model:       "claude-opus-4-7-high",
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
		TopP:        &topP,
		TopK:        &topK,
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "claude-opus-4-7", request.Model)
	requireOpus47AdaptiveThinking(t, request.Thinking)
	require.Nil(t, request.Temperature)
	require.Nil(t, request.TopP)
	require.Nil(t, request.TopK)

	outputConfig := requireOutputConfigMap(t, request.OutputConfig)
	require.Equal(t, "high", outputConfig["effort"])
}

func TestRequestOpenAI2ClaudeMessageOpus47ThinkingSuffixDefaultsHighEffort(t *testing.T) {
	request, err := RequestOpenAI2ClaudeMessage(nil, dto.GeneralOpenAIRequest{
		Model: "claude-opus-4-7-thinking",
		Messages: []dto.Message{
			{Role: "user", Content: "hello"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "claude-opus-4-7", request.Model)
	requireOpus47AdaptiveThinking(t, request.Thinking)

	outputConfig := requireOutputConfigMap(t, request.OutputConfig)
	require.Equal(t, "high", outputConfig["effort"])
}

func TestRequestOpenAI2ClaudeMessageOpus47ReasoningEffortPreservesOutputConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	request, err := RequestOpenAI2ClaudeMessage(ctx, dto.GeneralOpenAIRequest{
		Model:           "claude-opus-4-7",
		ReasoningEffort: "xhigh",
		OutputConfig: json.RawMessage(`{
			"format": {
				"type": "json_schema",
				"schema": {
					"type": "object",
					"properties": {
						"capital": {"type": "string"}
					},
					"required": ["capital"]
				}
			},
			"task_budget": "standard"
		}`),
		Messages: []dto.Message{
			{Role: "user", Content: "capital of France"},
		},
	})

	require.NoError(t, err)
	requireOpus47AdaptiveThinking(t, request.Thinking)

	outputConfig := requireOutputConfigMap(t, request.OutputConfig)
	require.Equal(t, "xhigh", outputConfig["effort"])
	require.Equal(t, "standard", outputConfig["task_budget"])

	format, ok := outputConfig["format"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "json_schema", format["type"])

	headers := http.Header{}
	CommonClaudeHeadersOperation(ctx, &headers, &relaycommon.RelayInfo{OriginModelName: "claude-opus-4-7"})
	require.Contains(t, headers.Get("anthropic-beta"), "task-budgets-2026-03-13")
}

func TestNormalizeOpus47RequestConvertsNativeEnabledThinking(t *testing.T) {
	budgetTokens := 4096
	temperature := 0.8
	topP := 0.95
	topK := 40
	request := &dto.ClaudeRequest{
		Model:       "claude-opus-4-7",
		Temperature: &temperature,
		TopP:        &topP,
		TopK:        &topK,
		Thinking: &dto.Thinking{
			Type:         "enabled",
			BudgetTokens: &budgetTokens,
		},
		OutputConfig: json.RawMessage(`{"format":{"type":"json_schema","schema":{"type":"object"}}}`),
	}

	require.NoError(t, NormalizeOpus47Request(request, "medium"))
	requireOpus47AdaptiveThinking(t, request.Thinking)
	require.Nil(t, request.Temperature)
	require.Nil(t, request.TopP)
	require.Nil(t, request.TopK)

	outputConfig := requireOutputConfigMap(t, request.OutputConfig)
	require.Equal(t, "medium", outputConfig["effort"])
	require.Contains(t, outputConfig, "format")
}

func TestNormalizeOpus47RequestMapsLegacyBudgetTokensToEffort(t *testing.T) {
	budgetTokens := 4096
	request := &dto.ClaudeRequest{
		Model: "claude-opus-4-7",
		Thinking: &dto.Thinking{
			Type:         "enabled",
			BudgetTokens: &budgetTokens,
		},
	}

	require.NoError(t, NormalizeOpus47Request(request, ""))
	requireOpus47AdaptiveThinking(t, request.Thinking)

	outputConfig := requireOutputConfigMap(t, request.OutputConfig)
	require.Equal(t, "high", outputConfig["effort"])
}
