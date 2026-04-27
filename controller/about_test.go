package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type aboutResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Legacy string `json:"legacy"`
		Config string `json:"config"`
	} `json:"data"`
}

type optionUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func setAboutOptionMap(t *testing.T, values map[string]string) {
	t.Helper()

	common.OptionMapRWMutex.Lock()
	previous := common.OptionMap
	common.OptionMap = values
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previous
		common.OptionMapRWMutex.Unlock()
	})
}

func performGetAbout(t *testing.T) aboutResponse {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	GetAbout(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response aboutResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func setupAboutOptionTestDB(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	common.OptionMapRWMutex.Lock()
	previousOptionMap := common.OptionMap
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()

	db, err := gorm.Open(sqlite.Open("file:about_option?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Option{}))
	model.DB = db

	t.Cleanup(func() {
		model.DB = previousDB
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func performUpdateAboutPageConfig(t *testing.T, value string) optionUpdateResponse {
	t.Helper()

	payload, err := common.Marshal(map[string]any{
		"key":   "AboutPageConfig",
		"value": value,
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/option", bytes.NewReader(payload))

	UpdateOption(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response optionUpdateResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestGetAboutReturnsLegacyAndStructuredConfig(t *testing.T) {
	setAboutOptionMap(t, map[string]string{
		"About":           "legacy about html",
		"AboutPageConfig": `{"sections":[{"title":"Overview"}]}`,
	})

	response := performGetAbout(t)

	require.True(t, response.Success)
	require.Equal(t, "legacy about html", response.Data.Legacy)
	require.Equal(t, `{"sections":[{"title":"Overview"}]}`, response.Data.Config)
}

func TestGetAboutHandlesMissingOptionValues(t *testing.T) {
	setAboutOptionMap(t, map[string]string{})

	response := performGetAbout(t)

	require.True(t, response.Success)
	require.Empty(t, response.Data.Legacy)
	require.Empty(t, response.Data.Config)
}

func TestUpdateOptionRejectsInvalidAboutPageConfigJSON(t *testing.T) {
	setupAboutOptionTestDB(t)

	for _, value := range []string{"{", "[]", "null"} {
		response := performUpdateAboutPageConfig(t, value)

		require.False(t, response.Success)
		require.True(t, strings.HasPrefix(response.Message, "AboutPageConfig JSON invalid: "), response.Message)
	}
}

func TestAboutPageConfigUsesAboutAuditAction(t *testing.T) {
	aboutMeta, ok := service.LookupSettingAuditMeta("About")
	require.True(t, ok)

	configMeta, ok := service.LookupSettingAuditMeta("AboutPageConfig")
	require.True(t, ok)
	require.Equal(t, aboutMeta, configMeta)
}
