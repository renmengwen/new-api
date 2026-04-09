package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type permissionGrant struct {
	Resource string
	Action   string
}

func grantPermissionActions(t *testing.T, db *gorm.DB, userId int, profileType string, actions ...permissionGrant) {
	t.Helper()

	profile := &model.PermissionProfile{
		ProfileName: profileType + "_profile_" + common.GetRandomString(6),
		ProfileType: profileType,
		Status:      model.CommonStatusEnabled,
		CreatedAtTs: common.GetTimestamp(),
		UpdatedAtTs: common.GetTimestamp(),
	}
	require.NoError(t, db.Create(profile).Error)

	for _, item := range actions {
		require.NoError(t, db.Create(&model.PermissionProfileItem{
			ProfileId:   profile.Id,
			ResourceKey: item.Resource,
			ActionKey:   item.Action,
			Allowed:     true,
			ScopeType:   model.ScopeTypeAll,
			CreatedAtTs: common.GetTimestamp(),
		}).Error)
	}

	require.NoError(t, db.Create(&model.UserPermissionBinding{
		UserId:        userId,
		ProfileId:     profile.Id,
		Status:        model.CommonStatusEnabled,
		EffectiveFrom: common.GetTimestamp(),
		CreatedAtTs:   common.GetTimestamp(),
	}).Error)
}
