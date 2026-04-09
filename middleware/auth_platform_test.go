package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminPlatformAuthAllowsAgentButAdminAuthStillBlocksAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	store := cookie.NewStore([]byte("test-secret"))
	router.Use(sessions.Sessions("test-session", store))

	router.GET("/login-agent", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", 1)
		session.Set("username", "agent-user")
		session.Set("role", common.RoleCommonUser)
		session.Set("user_type", "agent")
		session.Set("status", common.UserStatusEnabled)
		session.Set("group", "default")
		require.NoError(t, session.Save())
		c.Status(http.StatusOK)
	})

	router.GET("/admin-platform", AdminPlatformAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
	router.GET("/admin-legacy", AdminAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	loginRecorder := httptest.NewRecorder()
	loginRequest := httptest.NewRequest(http.MethodGet, "/login-agent", nil)
	router.ServeHTTP(loginRecorder, loginRequest)
	require.Equal(t, http.StatusOK, loginRecorder.Code)

	platformRecorder := httptest.NewRecorder()
	platformRequest := httptest.NewRequest(http.MethodGet, "/admin-platform", nil)
	platformRequest.Header.Set("New-Api-User", "1")
	for _, cookie := range loginRecorder.Result().Cookies() {
		platformRequest.AddCookie(cookie)
	}
	router.ServeHTTP(platformRecorder, platformRequest)
	require.Equal(t, http.StatusOK, platformRecorder.Code)
	require.Contains(t, platformRecorder.Body.String(), `"success":true`)

	legacyRecorder := httptest.NewRecorder()
	legacyRequest := httptest.NewRequest(http.MethodGet, "/admin-legacy", nil)
	legacyRequest.Header.Set("New-Api-User", "1")
	for _, cookie := range loginRecorder.Result().Cookies() {
		legacyRequest.AddCookie(cookie)
	}
	router.ServeHTTP(legacyRecorder, legacyRequest)
	require.Equal(t, http.StatusOK, legacyRecorder.Code)
	require.Contains(t, legacyRecorder.Body.String(), `"success":false`)
}
