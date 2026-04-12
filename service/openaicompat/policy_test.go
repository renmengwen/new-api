package openaicompat

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func TestShouldChatCompletionsUseResponsesPolicy(t *testing.T) {
	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled: true,
		ChannelTypes: []int{
			constant.ChannelTypeVolcEngine,
		},
		ModelPatterns: []string{
			"^doubao-seed-translation-.*$",
			"^doubao-seed-1-6-thinking-.*$",
		},
	}

	t.Run("volcengine translation model", func(t *testing.T) {
		model := "doubao-seed-translation-en-250101"
		if !common.IsVolcEngineResponsesModel(model) {
			t.Fatalf("test setup error: %q should be a VolcEngine Responses model", model)
		}
		if !ShouldChatCompletionsUseResponsesPolicy(policy, 0, constant.ChannelTypeVolcEngine, model) {
			t.Fatalf("ShouldChatCompletionsUseResponsesPolicy(...) = false, want true")
		}
	})

	t.Run("volcengine thinking model", func(t *testing.T) {
		model := "doubao-seed-1-6-thinking-250715"
		if !common.IsVolcEngineResponsesModel(model) {
			t.Fatalf("test setup error: %q should be a VolcEngine Responses model", model)
		}
		if !ShouldChatCompletionsUseResponsesPolicy(policy, 0, constant.ChannelTypeVolcEngine, model) {
			t.Fatalf("ShouldChatCompletionsUseResponsesPolicy(...) = false, want true")
		}
	})

	t.Run("ordinary doubao-pro-32k", func(t *testing.T) {
		model := "Doubao-pro-32k"
		if common.IsVolcEngineResponsesModel(model) {
			t.Fatalf("test setup error: %q should not be a VolcEngine Responses model", model)
		}
		if ShouldChatCompletionsUseResponsesPolicy(policy, 0, constant.ChannelTypeVolcEngine, model) {
			t.Fatalf("ShouldChatCompletionsUseResponsesPolicy(...) = true, want false")
		}
	})

	t.Run("non volcengine channel", func(t *testing.T) {
		model := "doubao-seed-translation-en-250101"
		if !common.IsVolcEngineResponsesModel(model) {
			t.Fatalf("test setup error: %q should be a VolcEngine Responses model", model)
		}
		if ShouldChatCompletionsUseResponsesPolicy(policy, 0, constant.ChannelTypeOpenAI, model) {
			t.Fatalf("ShouldChatCompletionsUseResponsesPolicy(...) = true, want false")
		}
	})
}
