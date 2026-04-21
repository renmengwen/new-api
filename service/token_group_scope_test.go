package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
)

func withTokenGroupScopeConfig(t *testing.T) {
	t.Helper()

	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalSpecialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.MarshalJSONString()

	t.Cleanup(func() {
		if err := setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups); err != nil {
			t.Fatalf("restore user usable groups: %v", err)
		}
		if err := types.LoadFromJsonString(
			ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup,
			originalSpecialGroups,
		); err != nil {
			t.Fatalf("restore group special usable groups: %v", err)
		}
	})

	if err := setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP","beta":"Beta"}`); err != nil {
		t.Fatalf("setup user usable groups: %v", err)
	}
	if err := types.LoadFromJsonString(
		ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup,
		`{}`,
	); err != nil {
		t.Fatalf("setup special usable groups: %v", err)
	}
}

func TestResolveUserTokenSelectableGroupsFallsBackToLegacyWhenDisabled(t *testing.T) {
	withTokenGroupScopeConfig(t)

	settingMap := dto.UserSetting{}
	groups := ResolveUserTokenSelectableGroups("vip", settingMap)
	legacyGroups := GetUserTokenSelectableGroups("vip")
	if len(groups) != len(legacyGroups) {
		t.Fatalf("expected legacy group count %d, got %d", len(legacyGroups), len(groups))
	}
	for name, desc := range legacyGroups {
		if groups[name] != desc {
			t.Fatalf("expected legacy group %s desc %q, got %q", name, desc, groups[name])
		}
	}
}

func TestResolveUserTokenSelectableGroupsUsesExplicitWhitelistWhenEnabled(t *testing.T) {
	withTokenGroupScopeConfig(t)

	settingMap := dto.UserSetting{}
	if err := common.UnmarshalJsonStr(`{"allowed_token_groups_enabled":true,"allowed_token_groups":["default","beta","auto"]}`, &settingMap); err != nil {
		t.Fatalf("unmarshal whitelist setting: %v", err)
	}

	groups := ResolveUserTokenSelectableGroups("default", settingMap)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %v", groups)
	}
	if groups["default"] != "Default" {
		t.Fatalf("expected default group to be preserved, got %v", groups)
	}
	if groups["beta"] != "Beta" {
		t.Fatalf("expected beta group to be preserved, got %v", groups)
	}
	if _, ok := groups["auto"]; !ok {
		t.Fatalf("expected explicit auto group to be preserved, got %v", groups)
	}
	if _, ok := groups["vip"]; ok {
		t.Fatalf("expected vip to be excluded from explicit whitelist, got %v", groups)
	}
}

func TestResolveAssignableTokenGroupsForAgentUsesOwnWhitelist(t *testing.T) {
	withTokenGroupScopeConfig(t)

	user := &model.User{
		Group:    "default",
		UserType: model.UserTypeAgent,
		Setting:  `{"allowed_token_groups_enabled":true,"allowed_token_groups":["default","beta"]}`,
	}

	assignable := ResolveAssignableTokenGroups(user)
	if len(assignable) != 2 {
		t.Fatalf("expected 2 assignable groups, got %v", assignable)
	}
	if _, ok := assignable["default"]; !ok {
		t.Fatalf("expected default to be assignable, got %v", assignable)
	}
	if _, ok := assignable["beta"]; !ok {
		t.Fatalf("expected beta to be assignable, got %v", assignable)
	}
	if _, ok := assignable["vip"]; ok {
		t.Fatalf("expected vip to be excluded from assignable groups, got %v", assignable)
	}
}

func TestValidateAllowedTokenGroupsAssignmentRejectsPrimaryGroupOutsideWhitelist(t *testing.T) {
	assignable := map[string]string{
		"default": "Default",
		"vip":     "VIP",
	}

	if _, err := ValidateAllowedTokenGroupsAssignment("default", true, []string{"vip"}, assignable); err == nil {
		t.Fatal("expected validation to reject primary group outside whitelist")
	}
}

func TestValidateAllowedTokenGroupsAssignmentAllowsLegacyPrimaryGroup(t *testing.T) {
	assignable := map[string]string{
		"default": "Default",
		"vip":     "VIP",
	}

	groups, err := ValidateAllowedTokenGroupsAssignment("legacy", true, []string{"legacy", "vip"}, assignable)
	if err != nil {
		t.Fatalf("expected legacy primary group to remain assignable, got %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 normalized groups, got %v", groups)
	}
}

func TestResolveUserTokenSelectableGroupsPreservesLegacyPrimaryGroupWhenWhitelisted(t *testing.T) {
	withTokenGroupScopeConfig(t)

	settingMap := dto.UserSetting{
		AllowedTokenGroupsEnabled: true,
		AllowedTokenGroups:        []string{"legacy", "beta"},
	}

	groups := ResolveUserTokenSelectableGroups("legacy", settingMap)
	if groups["legacy"] != "legacy" {
		t.Fatalf("expected legacy primary group to be preserved, got %v", groups)
	}
	if groups["beta"] != "Beta" {
		t.Fatalf("expected beta group to be preserved, got %v", groups)
	}
}
