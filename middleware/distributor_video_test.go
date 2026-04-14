package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetModelRequestForVideoGenerationList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/v1/video/generations", nil)

	modelRequest, shouldSelectChannel, err := getModelRequest(context)
	require.NoError(t, err)
	require.NotNil(t, modelRequest)
	require.False(t, shouldSelectChannel)

	relayMode, ok := context.Get("relay_mode")
	require.True(t, ok)
	require.Equal(t, relayconstant.RelayModeVideoFetchList, relayMode)
}

func TestGetModelRequestForVideoGenerationDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodDelete, "/v1/video/generations/task_123", nil)

	modelRequest, shouldSelectChannel, err := getModelRequest(context)
	require.NoError(t, err)
	require.NotNil(t, modelRequest)
	require.False(t, shouldSelectChannel)

	relayMode, ok := context.Get("relay_mode")
	require.True(t, ok)
	require.Equal(t, relayconstant.RelayModeVideoDelete, relayMode)
}
