package middleware

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func withPlaygroundGroupConfig(t *testing.T) {
	t.Helper()

	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalGroupRatios := ratio_setting.GroupRatio2JSONString()

	t.Cleanup(func() {
		if err := setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups); err != nil {
			t.Fatalf("restore user usable groups: %v", err)
		}
		if err := ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatios); err != nil {
			t.Fatalf("restore group ratios: %v", err)
		}
	})

	if err := setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`); err != nil {
		t.Fatalf("setup user usable groups: %v", err)
	}
	if err := ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":2,"beta":3}`); err != nil {
		t.Fatalf("setup group ratios: %v", err)
	}
}

func TestPlaygroundGroupAllowedUsesPrimaryGroupWhenRestrictionDisabled(t *testing.T) {
	withPlaygroundGroupConfig(t)

	user := &model.User{
		Group:  "default",
		Role:   common.RoleCommonUser,
		Status: common.UserStatusEnabled,
	}

	if !playgroundGroupAllowed(user, "default") {
		t.Fatal("expected primary group to be allowed")
	}
	if playgroundGroupAllowed(user, "vip") {
		t.Fatal("expected non-primary group to be rejected when restriction is disabled")
	}
}

func TestPlaygroundGroupAllowedUsesAllowedTokenGroupsWhenRestrictionEnabled(t *testing.T) {
	withPlaygroundGroupConfig(t)

	user := &model.User{
		Group:   "default",
		Role:    common.RoleCommonUser,
		Status:  common.UserStatusEnabled,
		Setting: `{"allowed_token_groups_enabled":true,"allowed_token_groups":["default","beta"]}`,
	}

	if !playgroundGroupAllowed(user, "beta") {
		t.Fatal("expected whitelisted token group to be allowed")
	}
	if playgroundGroupAllowed(user, "vip") {
		t.Fatal("expected non-whitelisted token group to be rejected")
	}
}
