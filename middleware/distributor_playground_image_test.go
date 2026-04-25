package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetModelRequestForPlaygroundImageGenerations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/pg/images/generations",
		strings.NewReader(`{"model":"gpt-image-2-plus","group":"TunnelForTL","prompt":"cat"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	modelRequest, shouldSelectChannel, err := getModelRequest(context)
	require.NoError(t, err)
	require.True(t, shouldSelectChannel)
	require.Equal(t, "gpt-image-2-plus", modelRequest.Model)
	require.Equal(t, "TunnelForTL", modelRequest.Group)
	require.Equal(t, "TunnelForTL", common.GetContextKeyString(context, constant.ContextKeyTokenGroup))

	relayMode, ok := context.Get("relay_mode")
	require.True(t, ok)
	require.Equal(t, relayconstant.RelayModeImagesGenerations, relayMode)
}
