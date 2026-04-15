package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionOverrideModelsAutoMigrate(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(
		&UserPermissionOverride{},
		&UserMenuOverride{},
		&UserDataScopeOverride{},
	))

	assert.True(t, DB.Migrator().HasTable(&UserPermissionOverride{}))
	assert.True(t, DB.Migrator().HasTable(&UserMenuOverride{}))
	assert.True(t, DB.Migrator().HasTable(&UserDataScopeOverride{}))
}
