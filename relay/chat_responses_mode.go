package relay

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func shouldUseResponsesForChatCompletionsWithPolicy(info *relaycommon.RelayInfo, modelMapping string, policy model_setting.ChatCompletionsToResponsesPolicy) bool {
	if info == nil || info.ChannelMeta == nil {
		return false
	}

	resolvedModel := info.OriginModelName
	if mappedModel, _, err := helper.ResolveMappedModelName(info.OriginModelName, modelMapping); err == nil && strings.TrimSpace(mappedModel) != "" {
		resolvedModel = mappedModel
	}

	if info.IsPlayground &&
		info.ChannelType == constant.ChannelTypeVolcEngine &&
		!strings.HasSuffix(resolvedModel, ratio_setting.CompactModelSuffix) &&
		common.IsVolcEngineResponsesModel(resolvedModel) {
		return true
	}

	return service.ShouldChatCompletionsUseResponsesPolicy(policy, info.ChannelId, info.ChannelType, resolvedModel)
}

func shouldUseResponsesForChatCompletions(info *relaycommon.RelayInfo, modelMapping string) bool {
	return shouldUseResponsesForChatCompletionsWithPolicy(
		info,
		modelMapping,
		model_setting.GetGlobalSettings().ChatCompletionsToResponsesPolicy,
	)
}
