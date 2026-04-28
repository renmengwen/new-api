package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func setupUserCacheRedis(t *testing.T) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	originalRDB := common.RDB
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = true
	common.RDB = redis.NewClient(&redis.Options{Addr: mr.Addr()})

	t.Cleanup(func() {
		_ = common.RDB.Close()
		common.RDB = originalRDB
		common.RedisEnabled = originalRedisEnabled
		mr.Close()
	})
}

func requireCachedEmail(t *testing.T, userId int, expected string) {
	t.Helper()

	userCache, err := model.GetUserCache(userId)
	require.NoError(t, err)
	require.Equal(t, expected, userCache.Email)
	require.Eventually(t, func() bool {
		var cached model.UserBase
		if err := common.RedisHGetObj(fmt.Sprintf("user:%d", userId), &cached); err != nil {
			return false
		}
		return cached.Email == expected
	}, time.Second, 10*time.Millisecond)
}

func TestUpdateAdminUserWithOperatorInvalidatesUserCache(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.AgentUserRelation{},
		&model.AgentQuotaPolicy{},
		&model.AdminAuditLog{},
		&model.QuotaAccount{},
		&model.QuotaTransferOrder{},
		&model.QuotaLedger{},
		&model.Log{},
	))
	setupUserCacheRedis(t)

	user := model.User{
		Username:    "cache_admin_user",
		Password:    "hashed-password",
		DisplayName: "Cache User",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeEndUser,
		Group:       "default",
		Email:       "before-cache@example.com",
		Quota:       100,
		Remark:      "before",
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, common.RedisHSetObj(fmt.Sprintf("user:%d", user.Id), user.ToBaseUser(), time.Duration(common.RedisKeyCacheSeconds())*time.Second))

	err := UpdateAdminUserWithOperator(user.Id, UpdateAdminUserRequest{
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Email:       "after-cache@example.com",
		Group:       user.Group,
		Remark:      user.Remark,
		Quota:       user.Quota,
	}, 0, common.RoleRootUser, "127.0.0.1")
	require.NoError(t, err)

	requireCachedEmail(t, user.Id, "after-cache@example.com")
}

func TestUpdateAgentWithOperatorInvalidatesUserCache(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.AgentProfile{},
		&model.AdminAuditLog{},
	))
	setupUserCacheRedis(t)

	user := model.User{
		Username:    "cache_agent_user",
		Password:    "hashed-password",
		DisplayName: "Cache Agent",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       "default",
		Email:       "before-agent-cache@example.com",
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.AgentProfile{
		UserId:       user.Id,
		AgentName:    "Cache Agent",
		CompanyName:  "Before Co",
		ContactPhone: "13800000000",
		Status:       model.CommonStatusEnabled,
		CreatedAtTs:  common.GetTimestamp(),
		UpdatedAtTs:  common.GetTimestamp(),
	}).Error)
	require.NoError(t, common.RedisHSetObj(fmt.Sprintf("user:%d", user.Id), user.ToBaseUser(), time.Duration(common.RedisKeyCacheSeconds())*time.Second))

	err := UpdateAgentWithOperator(user.Id, UpdateAgentRequest{
		DisplayName:  user.DisplayName,
		AgentName:    "Cache Agent",
		CompanyName:  "After Co",
		ContactPhone: "13900000000",
		Email:        "after-agent-cache@example.com",
		Group:        user.Group,
	}, 0, "", "127.0.0.1")
	require.NoError(t, err)

	requireCachedEmail(t, user.Id, "after-agent-cache@example.com")
}

func TestUpdateAdminManagerWithOperatorInvalidatesUserCache(t *testing.T) {
	db := setupAdminPermissionServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.AdminAuditLog{}))
	setupUserCacheRedis(t)

	user := model.User{
		Username:    "cache_admin_manager",
		Password:    "hashed-password",
		DisplayName: "Cache Admin",
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       "default",
		Email:       "before-manager-cache@example.com",
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, common.RedisHSetObj(fmt.Sprintf("user:%d", user.Id), user.ToBaseUser(), time.Duration(common.RedisKeyCacheSeconds())*time.Second))

	err := UpdateAdminManagerWithOperator(user.Id, UpdateAdminManagerRequest{
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Email:       "after-manager-cache@example.com",
	}, 0, model.UserTypeRoot, "127.0.0.1")
	require.NoError(t, err)

	requireCachedEmail(t, user.Id, "after-manager-cache@example.com")
}
