package doubao

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConvertToRequestPayloadAddsReferenceRolesForSeedanceReferences(t *testing.T) {
	var req relaycommon.TaskSubmitReq
	err := common.Unmarshal([]byte(`{
		"model": "doubao-seedance-2-0-260128",
		"prompt": "test prompt",
		"images": [
			"https://example.com/reference-1.png",
			"https://example.com/reference-2.png"
		],
		"metadata": {
			"videos": [
				"https://example.com/reference.mp4"
			],
			"audio_url": [
				"https://example.com/reference.mp3"
			],
			"ratio": "16:9",
			"duration": 11,
			"watermark": false
		}
	}`), &req)
	require.NoError(t, err)

	adaptor := &TaskAdaptor{}
	payload, err := adaptor.convertToRequestPayload(&req)
	require.NoError(t, err)
	require.Len(t, payload.Content, 5)

	require.Equal(t, "text", payload.Content[0].Type)
	require.Equal(t, "test prompt", payload.Content[0].Text)

	require.Equal(t, "image_url", payload.Content[1].Type)
	require.Equal(t, "reference_image", payload.Content[1].Role)
	require.NotNil(t, payload.Content[1].ImageURL)
	require.Equal(t, "https://example.com/reference-1.png", payload.Content[1].ImageURL.URL)

	require.Equal(t, "image_url", payload.Content[2].Type)
	require.Equal(t, "reference_image", payload.Content[2].Role)
	require.NotNil(t, payload.Content[2].ImageURL)
	require.Equal(t, "https://example.com/reference-2.png", payload.Content[2].ImageURL.URL)

	require.Equal(t, "video_url", payload.Content[3].Type)
	require.Equal(t, "reference_video", payload.Content[3].Role)
	require.NotNil(t, payload.Content[3].VideoURL)
	require.Equal(t, "https://example.com/reference.mp4", payload.Content[3].VideoURL.URL)

	require.Equal(t, "audio_url", payload.Content[4].Type)
	require.Equal(t, "reference_audio", payload.Content[4].Role)
	require.NotNil(t, payload.Content[4].AudioURL)
	require.Equal(t, "https://example.com/reference.mp3", payload.Content[4].AudioURL.URL)
}

func TestConvertToRequestPayloadKeepsSingleImageWithoutRoleForI2V(t *testing.T) {
	var req relaycommon.TaskSubmitReq
	err := common.Unmarshal([]byte(`{
		"model": "doubao-seedance-1-5-pro-251215",
		"prompt": "single frame prompt",
		"images": [
			"https://example.com/first-frame.png"
		],
		"metadata": {
			"ratio": "adaptive",
			"duration": 5,
			"watermark": false
		}
	}`), &req)
	require.NoError(t, err)

	adaptor := &TaskAdaptor{}
	payload, err := adaptor.convertToRequestPayload(&req)
	require.NoError(t, err)
	require.Len(t, payload.Content, 2)

	require.Equal(t, "text", payload.Content[0].Type)
	require.Equal(t, "image_url", payload.Content[1].Type)
	require.Empty(t, payload.Content[1].Role)
	require.NotNil(t, payload.Content[1].ImageURL)
	require.Equal(t, "https://example.com/first-frame.png", payload.Content[1].ImageURL.URL)
}

func TestConvertToRequestPayloadUsesExplicitImageRoles(t *testing.T) {
	var req relaycommon.TaskSubmitReq
	err := common.Unmarshal([]byte(`{
		"model": "doubao-seedance-1-5-pro-251215",
		"prompt": "first and last frame prompt",
		"images": [
			"https://example.com/first-frame.png",
			"https://example.com/last-frame.png"
		],
		"metadata": {
			"image_roles": [
				"first_frame",
				"last_frame"
			],
			"ratio": "adaptive",
			"duration": 5,
			"watermark": false
		}
	}`), &req)
	require.NoError(t, err)

	adaptor := &TaskAdaptor{}
	payload, err := adaptor.convertToRequestPayload(&req)
	require.NoError(t, err)
	require.Len(t, payload.Content, 3)

	require.Equal(t, "image_url", payload.Content[1].Type)
	require.Equal(t, "first_frame", payload.Content[1].Role)
	require.NotNil(t, payload.Content[1].ImageURL)
	require.Equal(t, "https://example.com/first-frame.png", payload.Content[1].ImageURL.URL)

	require.Equal(t, "image_url", payload.Content[2].Type)
	require.Equal(t, "last_frame", payload.Content[2].Role)
	require.NotNil(t, payload.Content[2].ImageURL)
	require.Equal(t, "https://example.com/last-frame.png", payload.Content[2].ImageURL.URL)
}

func TestValidateRequestAndSetActionMarksPromptOnlySeedanceAsTextGenerate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(`{
		"model": "doubao-seedance-2-0-260128",
		"prompt": "text to video prompt",
		"size": "1280x720",
		"duration": 5,
		"metadata": {
			"aspect_ratio": "16:9",
			"resolution": "720p",
			"input_video": false
		}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	adaptor := &TaskAdaptor{}

	taskErr := adaptor.ValidateRequestAndSetAction(c, info)
	require.Nil(t, taskErr)
	require.Equal(t, constant.TaskActionTextGenerate, info.Action)
}

func TestValidateRequestAndSetActionMarksReferenceInputsAsGenerate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(`{
		"model": "doubao-seedance-2-0-260128",
		"prompt": "reference video prompt",
		"metadata": {
			"videos": ["https://example.com/reference.mp4"]
		}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	adaptor := &TaskAdaptor{}

	taskErr := adaptor.ValidateRequestAndSetAction(c, info)
	require.Nil(t, taskErr)
	require.Equal(t, constant.TaskActionGenerate, info.Action)
}
