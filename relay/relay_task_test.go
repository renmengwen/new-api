package relay

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestApplyEstimatedTaskOtherRatiosSkipsAdvancedMediaLegacyEstimateBilling(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			BillingMode:      types.BillingModeAdvanced,
			AdvancedRuleType: types.AdvancedRuleTypeMediaTask,
			Quota:            4400,
		},
	}

	applyEstimatedTaskOtherRatios(info, "advanced-media-model", map[string]float64{
		"seconds":    5,
		"resolution": 1.5,
	})

	require.Equal(t, 4400, info.PriceData.Quota)
	require.Nil(t, info.PriceData.OtherRatios)
}

func TestApplyEstimatedTaskOtherRatiosKeepsLegacyEstimateBillingForPerRequestTasks(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			BillingMode: types.BillingModePerRequest,
			Quota:       1000,
		},
	}

	applyEstimatedTaskOtherRatios(info, "legacy-task-model", map[string]float64{
		"seconds":    5,
		"resolution": 1.5,
	})

	require.Equal(t, 7500, info.PriceData.Quota)
	require.Equal(t, map[string]float64{
		"seconds":    5,
		"resolution": 1.5,
	}, info.PriceData.OtherRatios)
}

func TestApplyAdjustedTaskOtherRatiosSkipsAdvancedMediaLegacySubmitAdjustment(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			BillingMode:      types.BillingModeAdvanced,
			AdvancedRuleType: types.AdvancedRuleTypeMediaTask,
			Quota:            4400,
		},
	}

	finalQuota := applyAdjustedTaskOtherRatios(info, map[string]float64{
		"seconds":    5,
		"resolution": 1.5,
	})

	require.Equal(t, 4400, finalQuota)
	require.Equal(t, 4400, info.PriceData.Quota)
	require.Nil(t, info.PriceData.OtherRatios)
}

func TestApplyAdjustedTaskOtherRatiosKeepsLegacySubmitAdjustmentForPerRequestTasks(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			BillingMode: types.BillingModePerRequest,
			Quota:       1000,
		},
	}

	finalQuota := applyAdjustedTaskOtherRatios(info, map[string]float64{
		"seconds":    5,
		"resolution": 1.5,
	})

	require.Equal(t, 7500, finalQuota)
	require.Equal(t, 7500, info.PriceData.Quota)
	require.Equal(t, map[string]float64{
		"seconds":    5,
		"resolution": 1.5,
	}, info.PriceData.OtherRatios)
}
