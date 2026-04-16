package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestCreateUserPersistsRequestedGroup(t *testing.T) {
	db := setupUserValidationTestDB(t)

	ctx, recorder := newCreateUserContext(t, map[string]any{
		"username": "legacy-group-user",
		"password": "12345678",
		"group":    "EZModel",
	})

	CreateUser(ctx)

	var response userValidationAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	var user model.User
	require.NoError(t, db.Where("username = ?", "legacy-group-user").First(&user).Error)
	require.Equal(t, "EZModel", user.Group)
}
