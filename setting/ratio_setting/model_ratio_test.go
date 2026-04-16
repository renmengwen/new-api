package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClaudeCompletionRatioDefaultsRemainButAreNotLocked(t *testing.T) {
	info := GetCompletionRatioInfo("claude-sonnet-4-5-20250929-thinking")
	require.Equal(t, 5.0, info.Ratio)
	require.False(t, info.Locked)
	require.Equal(t, 5.0, GetCompletionRatio("claude-sonnet-4-5-20250929-thinking"))
}

func TestNonClaudeLockedCompletionRatioRemainsLocked(t *testing.T) {
	info := GetCompletionRatioInfo("gpt-5.4")
	require.Equal(t, 6.0, info.Ratio)
	require.True(t, info.Locked)
}
