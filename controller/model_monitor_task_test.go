package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotifyModelMonitorFailuresEmailsAdmins(t *testing.T) {
	originalNotify := notifyModelMonitorAdminsByEmail
	t.Cleanup(func() {
		notifyModelMonitorAdminsByEmail = originalNotify
	})

	var gotType string
	var gotSubject string
	var gotContent string
	notifyModelMonitorAdminsByEmail = func(t string, subject string, content string) {
		gotType = t
		gotSubject = subject
		gotContent = content
	}

	notifyModelMonitorFailures(3, []modelMonitorFailureNotificationItem{
		{
			Model:               "claude-opus-4-7",
			ChannelID:           12,
			ChannelName:         "Claude primary",
			Status:              modelMonitorChannelStatusTimeout,
			Error:               "timeout",
			ConsecutiveFailures: 3,
		},
	})

	require.Equal(t, "model_monitor", gotType)
	require.Equal(t, "Model monitor failure threshold reached", gotSubject)
	require.Contains(t, gotContent, "threshold=3")
	require.Contains(t, gotContent, "model=claude-opus-4-7")
	require.Contains(t, gotContent, "channel_id=12")
	require.Contains(t, gotContent, "error=timeout")
	require.False(t, strings.Contains(gotContent, "omitted_targets"))
}
