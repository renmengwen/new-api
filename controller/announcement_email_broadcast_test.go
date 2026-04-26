package controller

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type announcementEmailBroadcastAPIResponse struct {
	Success bool                                     `json:"success"`
	Message string                                   `json:"message"`
	Data    service.AnnouncementEmailBroadcastResult `json:"data"`
}

func newAnnouncementEmailBroadcastContext(t *testing.T, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	payload, err := common.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/notice/email-broadcast", io.NopCloser(bytes.NewReader(payload)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	return ctx, recorder
}

func TestBroadcastAnnouncementEmailControllerReturnsResult(t *testing.T) {
	setupUserSettingTestDB(t)

	ctx, recorder := newAnnouncementEmailBroadcastContext(t, service.AnnouncementEmailBroadcastRequest{
		Source:  service.AnnouncementEmailSourceNotice,
		Target:  service.AnnouncementEmailTargetAll,
		Title:   "Subject",
		Content: "<p>Body</p>",
	})

	BroadcastAnnouncementEmail(ctx)

	var response announcementEmailBroadcastAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.True(t, response.Success, response.Message)
	require.Equal(t, 0, response.Data.SentCount)
	require.Equal(t, 0, response.Data.SkippedCount)
	require.Equal(t, 0, response.Data.FailedCount)
}

func TestBroadcastAnnouncementEmailControllerRejectsInvalidJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/notice/email-broadcast", bytes.NewReader([]byte(`{`)))
	ctx.Request.Header.Set("Content-Type", "application/json")

	BroadcastAnnouncementEmail(ctx)

	var response announcementEmailBroadcastAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.False(t, response.Success)
}
