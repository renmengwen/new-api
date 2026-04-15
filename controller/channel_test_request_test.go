package controller

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestDefaultChannelTestImageSize(t *testing.T) {
	t.Parallel()

	volcengineChannel := &model.Channel{Type: constant.ChannelTypeVolcEngine}
	if got := defaultChannelTestImageSize("doubao-seedream-5-0-260128", volcengineChannel); got != "2048x2048" {
		t.Fatalf("defaultChannelTestImageSize() = %s, want 2048x2048", got)
	}

	openAIChannel := &model.Channel{Type: constant.ChannelTypeOpenAI}
	if got := defaultChannelTestImageSize("gpt-image-1", openAIChannel); got != "1024x1024" {
		t.Fatalf("defaultChannelTestImageSize() = %s, want 1024x1024", got)
	}
}

func TestNormalizeChannelTestEndpointForVolcEngineResponsesModel(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Type: constant.ChannelTypeVolcEngine}

	got := normalizeChannelTestEndpoint(channel, "doubao-seed-translation-250915", "")
	if got != string(constant.EndpointTypeOpenAIResponse) {
		t.Fatalf("normalizeChannelTestEndpoint() = %q, want %q", got, constant.EndpointTypeOpenAIResponse)
	}
}

func TestNormalizeChannelTestEndpointForVolcEngineThinkingResponsesModel(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Type: constant.ChannelTypeVolcEngine}

	got := normalizeChannelTestEndpoint(channel, "doubao-seed-1-6-thinking-250715", "")
	if got != string(constant.EndpointTypeOpenAIResponse) {
		t.Fatalf("normalizeChannelTestEndpoint() = %q, want %q", got, constant.EndpointTypeOpenAIResponse)
	}
}

func TestResolveChannelTestModelNameUsesMappedAlias(t *testing.T) {
	t.Parallel()

	modelMapping := `{"alias-translation":"doubao-seed-translation-250915"}`
	channel := &model.Channel{
		Type:         constant.ChannelTypeVolcEngine,
		ModelMapping: &modelMapping,
	}

	got := resolveChannelTestModelName(channel, "alias-translation")
	if got != "doubao-seed-translation-250915" {
		t.Fatalf("resolveChannelTestModelName() = %q, want doubao-seed-translation-250915", got)
	}
}

func TestNormalizeChannelTestEndpointForMappedVolcEngineResponsesAlias(t *testing.T) {
	t.Parallel()

	modelMapping := `{"alias-translation":"doubao-seed-translation-250915"}`
	channel := &model.Channel{
		Type:         constant.ChannelTypeVolcEngine,
		ModelMapping: &modelMapping,
	}

	got := normalizeChannelTestEndpoint(channel, resolveChannelTestModelName(channel, "alias-translation"), "")
	if got != string(constant.EndpointTypeOpenAIResponse) {
		t.Fatalf("normalizeChannelTestEndpoint() = %q, want %q", got, constant.EndpointTypeOpenAIResponse)
	}
}

func TestNormalizeChannelTestEndpointPrefersCompactForVolcEngineResponsesModel(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Type: constant.ChannelTypeVolcEngine}

	modelName := ratio_setting.WithCompactModelSuffix("doubao-seed-translation-250915")
	got := normalizeChannelTestEndpoint(channel, modelName, "")
	if got != string(constant.EndpointTypeOpenAIResponseCompact) {
		t.Fatalf("normalizeChannelTestEndpoint() = %q, want %q", got, constant.EndpointTypeOpenAIResponseCompact)
	}
}

func TestBuildTestRequestForVolcEngineResponsesModelUsesTypedContent(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Type: constant.ChannelTypeVolcEngine}

	request := buildTestRequest("doubao-seed-translation-250915", string(constant.EndpointTypeOpenAIResponse), channel, false)
	responsesRequest, ok := request.(*dto.OpenAIResponsesRequest)
	if !ok {
		t.Fatalf("buildTestRequest() returned %T, want *dto.OpenAIResponsesRequest", request)
	}

	var input []map[string]any
	if err := common.Unmarshal(responsesRequest.Input, &input); err != nil {
		t.Fatalf("failed to unmarshal responses input: %v", err)
	}
	if len(input) != 1 {
		t.Fatalf("len(input) = %d, want 1", len(input))
	}

	content, ok := input[0]["content"].([]any)
	if !ok {
		t.Fatalf("content type = %T, want []any", input[0]["content"])
	}
	if len(content) != 1 {
		t.Fatalf("len(content) = %d, want 1", len(content))
	}

	item, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] type = %T, want map[string]any", content[0])
	}
	if item["type"] != "input_text" {
		t.Fatalf("content[0].type = %v, want input_text", item["type"])
	}
	if item["text"] != "hi" {
		t.Fatalf("content[0].text = %v, want hi", item["text"])
	}
}

func TestBuildTestRequestForMappedVolcEngineResponsesAliasUsesTypedContent(t *testing.T) {
	t.Parallel()

	modelMapping := `{"alias-translation":"doubao-seed-translation-250915"}`
	channel := &model.Channel{
		Type:         constant.ChannelTypeVolcEngine,
		ModelMapping: &modelMapping,
	}

	request := buildTestRequest("alias-translation", string(constant.EndpointTypeOpenAIResponse), channel, false)
	responsesRequest, ok := request.(*dto.OpenAIResponsesRequest)
	if !ok {
		t.Fatalf("buildTestRequest() returned %T, want *dto.OpenAIResponsesRequest", request)
	}

	var input []map[string]any
	if err := common.Unmarshal(responsesRequest.Input, &input); err != nil {
		t.Fatalf("failed to unmarshal responses input: %v", err)
	}

	content, ok := input[0]["content"].([]any)
	if !ok {
		t.Fatalf("content type = %T, want []any", input[0]["content"])
	}
	if len(content) != 1 {
		t.Fatalf("len(content) = %d, want 1", len(content))
	}

	item, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] type = %T, want map[string]any", content[0])
	}
	if item["type"] != "input_text" {
		t.Fatalf("content[0].type = %v, want input_text", item["type"])
	}
}

func TestNormalizeChannelTestEndpointKeepsVolcEngineChatModelDefault(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Type: constant.ChannelTypeVolcEngine}

	got := normalizeChannelTestEndpoint(channel, "Doubao-pro-32k", "")
	if got != "" {
		t.Fatalf("normalizeChannelTestEndpoint() = %q, want empty string", got)
	}
}

func TestChannelRejectsVolcEngineSeedanceModel(t *testing.T) {
	t.Parallel()

	channel := &model.Channel{Type: constant.ChannelTypeVolcEngine}

	result := testChannel(channel, "doubao-seedance-1-5-pro", "", false)
	if result.localErr == nil {
		t.Fatal("testChannel() localErr is nil, want unsupported error")
	}
	if !strings.Contains(strings.ToLower(result.localErr.Error()), "not supported") {
		t.Fatalf("testChannel() localErr = %q, want unsupported error", result.localErr.Error())
	}
}

func TestChannelRejectsMappedVolcEngineSeedanceAlias(t *testing.T) {
	t.Parallel()

	modelMapping := `{"alias-seedance":"doubao-seedance-2-0"}`
	channel := &model.Channel{
		Type:         constant.ChannelTypeVolcEngine,
		ModelMapping: &modelMapping,
	}

	result := testChannel(channel, "alias-seedance", "", false)
	if result.localErr == nil {
		t.Fatal("testChannel() localErr is nil, want unsupported error")
	}
	if !strings.Contains(strings.ToLower(result.localErr.Error()), "not supported") {
		t.Fatalf("testChannel() localErr = %q, want unsupported error", result.localErr.Error())
	}
}
