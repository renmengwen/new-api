package router

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApiRouterRegistersNoticeEmailBroadcastAsRootRoute(t *testing.T) {
	source, err := os.ReadFile("api-router.go")
	require.NoError(t, err)

	route := `apiRouter.POST("/notice/email-broadcast", middleware.RootAuth(), controller.BroadcastAnnouncementEmail)`
	require.True(t, strings.Contains(string(source), route), "notice email broadcast route must be root-only")
}
