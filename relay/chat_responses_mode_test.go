package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestShouldUseResponsesForChatCompletionsWithPolicy(t *testing.T) {
	t.Parallel()

	policy := model_setting.ChatCompletionsToResponsesPolicy{
		Enabled:      true,
		ChannelTypes: []int{constant.ChannelTypeVolcEngine},
		ModelPatterns: []string{
			"^doubao-seed-translation-.*$",
			"^doubao-seed-1-6-thinking-.*$",
		},
	}

	tests := []struct {
		name         string
		info         relaycommon.RelayInfo
		modelMapping string
		policy       model_setting.ChatCompletionsToResponsesPolicy
		want         bool
	}{
		{
			name: "playground translation model uses responses",
			info: relaycommon.RelayInfo{
				OriginModelName: "doubao-seed-translation-250915",
				IsPlayground:    true,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelId:   1,
					ChannelType: constant.ChannelTypeVolcEngine,
				},
			},
			policy: policy,
			want:   true,
		},
		{
			name: "playground mapped alias uses responses",
			info: relaycommon.RelayInfo{
				OriginModelName: "alias-translation",
				IsPlayground:    true,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelId:   1,
					ChannelType: constant.ChannelTypeVolcEngine,
				},
			},
			modelMapping: `{"alias-translation":"doubao-seed-translation-250915"}`,
			policy:       policy,
			want:         true,
		},
		{
			name: "playground normal volcengine chat model stays chat",
			info: relaycommon.RelayInfo{
				OriginModelName: "Doubao-pro-32k",
				IsPlayground:    true,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelId:   1,
					ChannelType: constant.ChannelTypeVolcEngine,
				},
			},
			policy: policy,
			want:   false,
		},
		{
			name: "non playground still follows explicit policy with mapped model",
			info: relaycommon.RelayInfo{
				OriginModelName: "alias-thinking",
				IsPlayground:    false,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelId:   1,
					ChannelType: constant.ChannelTypeVolcEngine,
				},
			},
			modelMapping: `{"alias-thinking":"doubao-seed-1-6-thinking-250715"}`,
			policy:       policy,
			want:         true,
		},
		{
			name: "compact suffix is not auto promoted by playground whitelist without policy",
			info: relaycommon.RelayInfo{
				OriginModelName: ratio_setting.WithCompactModelSuffix("doubao-seed-translation-250915"),
				IsPlayground:    true,
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelId:   1,
					ChannelType: constant.ChannelTypeVolcEngine,
				},
			},
			policy: model_setting.ChatCompletionsToResponsesPolicy{},
			want:   false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := shouldUseResponsesForChatCompletionsWithPolicy(&tt.info, tt.modelMapping, tt.policy); got != tt.want {
				t.Fatalf("shouldUseResponsesForChatCompletionsWithPolicy() = %t, want %t", got, tt.want)
			}
		})
	}
}
