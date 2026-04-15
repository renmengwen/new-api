package helper

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	jsoncommon "github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

func ResolveMappedModelName(originModelName, modelMapping string) (string, bool, error) {
	if modelMapping == "" || modelMapping == "{}" {
		return originModelName, false, nil
	}

	modelMap := make(map[string]string)
	if err := jsoncommon.UnmarshalJsonStr(modelMapping, &modelMap); err != nil {
		return "", false, fmt.Errorf("unmarshal_model_mapping_failed")
	}

	currentModel := originModelName
	visitedModels := map[string]bool{
		currentModel: true,
	}
	for {
		mappedModel, exists := modelMap[currentModel]
		if !exists || mappedModel == "" {
			break
		}
		if visitedModels[mappedModel] {
			if mappedModel == currentModel {
				return currentModel, currentModel != originModelName, nil
			}
			return "", false, errors.New("model_mapping_contains_cycle")
		}
		visitedModels[mappedModel] = true
		currentModel = mappedModel
	}

	return currentModel, currentModel != originModelName, nil
}

func ModelMappedHelper(c *gin.Context, info *relaycommon.RelayInfo, request dto.Request) error {
	if info.ChannelMeta == nil {
		info.ChannelMeta = &relaycommon.ChannelMeta{}
	}

	isResponsesCompact := info.RelayMode == relayconstant.RelayModeResponsesCompact
	originModelName := info.OriginModelName
	mappingModelName := originModelName
	if isResponsesCompact && strings.HasSuffix(originModelName, ratio_setting.CompactModelSuffix) {
		mappingModelName = strings.TrimSuffix(originModelName, ratio_setting.CompactModelSuffix)
	}

	// map model name
	modelMapping := c.GetString("model_mapping")
	if modelMapping != "" && modelMapping != "{}" {
		resolvedModel, isMapped, err := ResolveMappedModelName(mappingModelName, modelMapping)
		if err != nil {
			return err
		}
		info.IsModelMapped = isMapped
		if info.IsModelMapped {
			info.UpstreamModelName = resolvedModel
		}
	}

	if isResponsesCompact {
		finalUpstreamModelName := mappingModelName
		if info.IsModelMapped && info.UpstreamModelName != "" {
			finalUpstreamModelName = info.UpstreamModelName
		}
		info.UpstreamModelName = finalUpstreamModelName
		info.OriginModelName = ratio_setting.WithCompactModelSuffix(finalUpstreamModelName)
	}
	if request != nil {
		request.SetModelName(info.UpstreamModelName)
	}
	return nil
}
