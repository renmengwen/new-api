package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestUserEditRefreshesCachedEmailAfterEmailUpdate(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	originalRDB := common.RDB
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = true
	common.RDB = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = common.RDB.Close()
		common.RDB = originalRDB
		common.RedisEnabled = originalRedisEnabled
	})

	username := fmt.Sprintf("edit_email_%d", time.Now().UnixNano())
	user := User{
		Username:    username,
		Password:    "hashed-password",
		DisplayName: "before name",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    UserTypeEndUser,
		Group:       "default",
		Email:       "before@example.com",
		Quota:       10,
		Remark:      "before remark",
	}
	require.NoError(t, DB.Create(&user).Error)
	t.Cleanup(func() {
		DB.Delete(&User{}, user.Id)
	})
	require.NoError(t, common.RedisHSetObj(getUserCacheKey(user.Id), user.ToBaseUser(), time.Duration(common.RedisKeyCacheSeconds())*time.Second))

	user.Username = username + "_updated"
	user.Email = "after@example.com"
	user.DisplayName = "after name"
	user.Group = "vip"
	user.Quota = 20
	user.Remark = "after remark"

	require.NoError(t, user.Edit(false))

	userCache, err := GetUserCache(user.Id)
	require.NoError(t, err)
	require.Equal(t, "after@example.com", userCache.Email)
	require.Equal(t, username+"_updated", userCache.Username)
	require.Equal(t, "vip", userCache.Group)
	require.Equal(t, 20, userCache.Quota)
}
