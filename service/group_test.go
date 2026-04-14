package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
)

func TestGetUserTokenSelectableGroupsDoesNotAppendCurrentUserGroup(t *testing.T) {
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalSpecialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.MarshalJSONString()

	defer func() {
		if err := setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups); err != nil {
			t.Fatalf("restore user usable groups: %v", err)
		}
		if err := types.LoadFromJsonString(
			ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup,
			originalSpecialGroups,
		); err != nil {
			t.Fatalf("restore group special usable groups: %v", err)
		}
	}()

	if err := setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","group-z":"Group Z"}`); err != nil {
		t.Fatalf("setup user usable groups: %v", err)
	}
	if err := types.LoadFromJsonString(
		ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup,
		`{}`,
	); err != nil {
		t.Fatalf("setup special usable groups: %v", err)
	}

	tokenSelectableGroups := GetUserTokenSelectableGroups("group-y")
	if _, ok := tokenSelectableGroups["group-y"]; ok {
		t.Fatalf("token selectable groups should not append current user group, got %v", tokenSelectableGroups)
	}

	usableGroups := GetUserUsableGroups("group-y")
	if desc, ok := usableGroups["group-y"]; !ok || desc != "用户分组" {
		t.Fatalf("usable groups should append current user group with fallback description, got %v", usableGroups)
	}
}

func TestGetUserTokenSelectableGroupsKeepsSpecialOverrides(t *testing.T) {
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalSpecialGroups := ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup.MarshalJSONString()

	defer func() {
		if err := setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups); err != nil {
			t.Fatalf("restore user usable groups: %v", err)
		}
		if err := types.LoadFromJsonString(
			ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup,
			originalSpecialGroups,
		); err != nil {
			t.Fatalf("restore group special usable groups: %v", err)
		}
	}()

	if err := setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","beta":"Beta"}`); err != nil {
		t.Fatalf("setup user usable groups: %v", err)
	}
	if err := types.LoadFromJsonString(
		ratio_setting.GetGroupRatioSetting().GroupSpecialUsableGroup,
		`{"vip":{"-:default":"Default","+:premium":"Premium","special":"Special"}}`,
	); err != nil {
		t.Fatalf("setup special usable groups: %v", err)
	}

	groups := GetUserTokenSelectableGroups("vip")

	if _, ok := groups["default"]; ok {
		t.Fatalf("default group should be removed by special overrides, got %v", groups)
	}
	if groups["beta"] != "Beta" {
		t.Fatalf("existing configured group should be preserved, got %v", groups)
	}
	if groups["premium"] != "Premium" {
		t.Fatalf("prefixed added group should be preserved, got %v", groups)
	}
	if groups["special"] != "Special" {
		t.Fatalf("directly added group should be preserved, got %v", groups)
	}
	if _, ok := groups["vip"]; ok {
		t.Fatalf("token selectable groups should not append current user group fallback, got %v", groups)
	}
}
