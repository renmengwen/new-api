package volcengine

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"
)

func TestGetRequestURLUsesSpecialPlanOpenAIBaseForResponses(t *testing.T) {
	t.Parallel()

	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatOpenAIResponses,
		RelayMode:   relayconstant.RelayModeResponses,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelBaseUrl: "doubao-coding-plan",
		},
	}
	info.ChannelBaseUrl = "doubao-coding-plan"

	got, err := adaptor.GetRequestURL(info)
	if err != nil {
		t.Fatalf("GetRequestURL() error = %v", err)
	}

	const want = "https://ark.cn-beijing.volces.com/api/coding/v3/responses"
	if got != want {
		t.Fatalf("GetRequestURL() = %q, want %q", got, want)
	}
}
