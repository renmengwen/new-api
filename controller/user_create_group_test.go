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
		"email":    "legacy-group-user@example.com",
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

func TestCreateUserPersistsEmail(t *testing.T) {
	db := setupUserValidationTestDB(t)

	ctx, recorder := newCreateUserContext(t, map[string]any{
		"username": "legacy-email-user",
		"password": "12345678",
		"email":    "legacy-email-user@example.com",
		"group":    "default",
	})

	CreateUser(ctx)

	var response userValidationAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success, response.Message)

	var user model.User
	require.NoError(t, db.Where("username = ?", "legacy-email-user").First(&user).Error)
	require.Equal(t, "legacy-email-user@example.com", user.Email)
}

func TestCreateUserRejectsInvalidEmail(t *testing.T) {
	db := setupUserValidationTestDB(t)

	ctx, recorder := newCreateUserContext(t, map[string]any{
		"username": "legacy-bad-email",
		"password": "12345678",
		"email":    "not-an-email",
		"group":    "default",
	})

	CreateUser(ctx)

	var response userValidationAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)

	var count int64
	require.NoError(t, db.Model(&model.User{}).Where("username = ?", "legacy-bad-email").Count(&count).Error)
	require.Zero(t, count)
}

func TestCreateUserRejectsDuplicateEmail(t *testing.T) {
	db := setupUserValidationTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Username:    "legacy-dup-email-existing",
		Password:    "hashed-password",
		DisplayName: "Existing",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
		Email:       "legacy-duplicate@example.com",
	}).Error)

	ctx, recorder := newCreateUserContext(t, map[string]any{
		"username": "legacy-dup-email-new",
		"password": "12345678",
		"email":    "legacy-duplicate@example.com",
		"group":    "default",
	})

	CreateUser(ctx)

	var response userValidationAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.False(t, response.Success)

	var count int64
	require.NoError(t, db.Model(&model.User{}).Where("username = ?", "legacy-dup-email-new").Count(&count).Error)
	require.Zero(t, count)
}
