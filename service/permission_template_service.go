package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	permissionTemplateMenuResourcePrefix = "__menu__."
	permissionTemplateScopeActionKey     = "__scope__"
)

type PermissionTemplateItemInput struct {
	ResourceKey string `json:"resource_key"`
	ActionKey   string `json:"action_key"`
	Allowed     bool   `json:"allowed"`
}

type PermissionTemplateMenuItemInput struct {
	SectionKey string `json:"section_key"`
	ModuleKey  string `json:"module_key"`
	Allowed    bool   `json:"allowed"`
}

type PermissionTemplateDataScopeItemInput struct {
	ResourceKey string `json:"resource_key"`
	ScopeType   string `json:"scope_type"`
	ScopeValue  []int  `json:"scope_value"`
}

type PermissionTemplateUpsertRequest struct {
	ProfileName    string                                 `json:"profile_name"`
	ProfileType    string                                 `json:"profile_type"`
	Description    string                                 `json:"description"`
	Status         int                                    `json:"status"`
	Items          []PermissionTemplateItemInput          `json:"items"`
	MenuItems      []PermissionTemplateMenuItemInput      `json:"menu_items"`
	DataScopeItems []PermissionTemplateDataScopeItemInput `json:"data_scope_items"`
}

type PermissionTemplateDetail struct {
	Profile        model.PermissionProfile                `json:"profile"`
	Items          []model.PermissionProfileItem          `json:"items"`
	MenuItems      []PermissionTemplateMenuItemInput      `json:"menu_items"`
	DataScopeItems []PermissionTemplateDataScopeItemInput `json:"data_scope_items"`
}

func ListPermissionTemplates(pageInfo *common.PageInfo, profileType string) ([]model.PermissionProfile, int64, error) {
	return ListPermissionProfiles(pageInfo, profileType)
}

func GetPermissionTemplateDetail(profileId int) (*PermissionTemplateDetail, error) {
	var profile model.PermissionProfile
	if err := model.DB.First(&profile, profileId).Error; err != nil {
		return nil, err
	}

	var items []model.PermissionProfileItem
	if err := model.DB.Where("profile_id = ?", profileId).Order("id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	actionItems, menuItems, dataScopeItems := splitPermissionTemplateItems(items)

	return &PermissionTemplateDetail{
		Profile:        profile,
		Items:          actionItems,
		MenuItems:      menuItems,
		DataScopeItems: dataScopeItems,
	}, nil
}

func CreatePermissionTemplate(req PermissionTemplateUpsertRequest) (*PermissionTemplateDetail, error) {
	if err := validatePermissionTemplateRequest(req); err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	profile := model.PermissionProfile{
		ProfileName: req.ProfileName,
		ProfileType: req.ProfileType,
		Description: req.Description,
		Status:      normalizeTemplateStatus(req.Status),
		CreatedAtTs: now,
		UpdatedAtTs: now,
	}
	if err := tx.Create(&profile).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := replacePermissionTemplateItemsTx(tx, profile.Id, req, now); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return GetPermissionTemplateDetail(profile.Id)
}

func UpdatePermissionTemplate(profileId int, req PermissionTemplateUpsertRequest) (*PermissionTemplateDetail, error) {
	if err := validatePermissionTemplateRequest(req); err != nil {
		return nil, err
	}

	now := common.GetTimestamp()
	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var profile model.PermissionProfile
	if err := tx.First(&profile, profileId).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	profile.ProfileName = req.ProfileName
	profile.ProfileType = req.ProfileType
	profile.Description = req.Description
	profile.Status = normalizeTemplateStatus(req.Status)
	profile.UpdatedAtTs = now
	if err := tx.Save(&profile).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := replacePermissionTemplateItemsTx(tx, profile.Id, req, now); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return GetPermissionTemplateDetail(profile.Id)
}

func DeletePermissionTemplate(profileId int) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var profile model.PermissionProfile
		if err := permissionProfileForUpdateQuery(tx, profileId).First(&profile).Error; err != nil {
			return err
		}

		var activeBindingCount int64
		if err := tx.Model(&model.UserPermissionBinding{}).
			Where("profile_id = ? AND status = ?", profileId, model.CommonStatusEnabled).
			Count(&activeBindingCount).Error; err != nil {
			return err
		}
		if activeBindingCount > 0 {
			return fmt.Errorf("该模板正在被 %d 个账号使用，无法删除", activeBindingCount)
		}

		if err := tx.Where("profile_id = ?", profileId).Delete(&model.PermissionProfileItem{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&profile).Error; err != nil {
			return err
		}
		return nil
	})
}

func validatePermissionTemplateRequest(req PermissionTemplateUpsertRequest) error {
	if req.ProfileName == "" {
		return errors.New("profile_name is required")
	}
	switch req.ProfileType {
	case model.UserTypeAdmin, model.UserTypeAgent, model.UserTypeEndUser:
	default:
		return errors.New("unsupported profile_type")
	}
	for _, item := range req.Items {
		if item.ResourceKey == "" || item.ActionKey == "" {
			return errors.New("permission items must include resource_key and action_key")
		}
	}
	for _, item := range req.MenuItems {
		if strings.TrimSpace(item.SectionKey) == "" || strings.TrimSpace(item.ModuleKey) == "" {
			return errors.New("menu items must include section_key and module_key")
		}
	}
	for _, item := range req.DataScopeItems {
		if strings.TrimSpace(item.ResourceKey) == "" || strings.TrimSpace(item.ScopeType) == "" {
			return errors.New("data scope items must include resource_key and scope_type")
		}
	}
	return nil
}

func normalizeTemplateStatus(status int) int {
	if status == model.CommonStatusDisabled {
		return model.CommonStatusDisabled
	}
	return model.CommonStatusEnabled
}

func replacePermissionTemplateItemsTx(tx *gorm.DB, profileId int, req PermissionTemplateUpsertRequest, now int64) error {
	if err := tx.Where("profile_id = ?", profileId).Delete(&model.PermissionProfileItem{}).Error; err != nil {
		return err
	}
	if len(req.Items) == 0 && len(req.MenuItems) == 0 && len(req.DataScopeItems) == 0 {
		return nil
	}

	createItems := make([]model.PermissionProfileItem, 0, len(req.Items)+len(req.MenuItems)+len(req.DataScopeItems))
	for _, item := range req.Items {
		createItems = append(createItems, model.PermissionProfileItem{
			ProfileId:   profileId,
			ResourceKey: item.ResourceKey,
			ActionKey:   item.ActionKey,
			Allowed:     item.Allowed,
			ScopeType:   model.ScopeTypeAll,
			CreatedAtTs: now,
		})
	}
	for _, item := range req.MenuItems {
		if !item.Allowed {
			continue
		}
		createItems = append(createItems, model.PermissionProfileItem{
			ProfileId:   profileId,
			ResourceKey: permissionTemplateMenuResourcePrefix + item.SectionKey,
			ActionKey:   item.ModuleKey,
			Allowed:     true,
			ScopeType:   model.ScopeTypeAll,
			CreatedAtTs: now,
		})
	}
	for _, item := range req.DataScopeItems {
		scopeValueJSON := ""
		if len(item.ScopeValue) > 0 {
			data, err := common.Marshal(item.ScopeValue)
			if err != nil {
				return err
			}
			scopeValueJSON = string(data)
		}
		createItems = append(createItems, model.PermissionProfileItem{
			ProfileId:      profileId,
			ResourceKey:    item.ResourceKey,
			ActionKey:      permissionTemplateScopeActionKey,
			Allowed:        true,
			ScopeType:      item.ScopeType,
			ExtraScopeJSON: scopeValueJSON,
			CreatedAtTs:    now,
		})
	}
	return tx.Create(&createItems).Error
}

func splitPermissionTemplateItems(items []model.PermissionProfileItem) ([]model.PermissionProfileItem, []PermissionTemplateMenuItemInput, []PermissionTemplateDataScopeItemInput) {
	actionItems := make([]model.PermissionProfileItem, 0, len(items))
	menuItems := make([]PermissionTemplateMenuItemInput, 0)
	dataScopeItems := make([]PermissionTemplateDataScopeItemInput, 0)

	for _, item := range items {
		switch {
		case strings.HasPrefix(item.ResourceKey, permissionTemplateMenuResourcePrefix):
			menuItems = append(menuItems, PermissionTemplateMenuItemInput{
				SectionKey: strings.TrimPrefix(item.ResourceKey, permissionTemplateMenuResourcePrefix),
				ModuleKey:  item.ActionKey,
				Allowed:    item.Allowed,
			})
		case item.ActionKey == permissionTemplateScopeActionKey:
			dataScopeItem := PermissionTemplateDataScopeItemInput{
				ResourceKey: item.ResourceKey,
				ScopeType:   item.ScopeType,
				ScopeValue:  []int{},
			}
			if item.ExtraScopeJSON != "" {
				_ = common.UnmarshalJsonStr(item.ExtraScopeJSON, &dataScopeItem.ScopeValue)
			}
			dataScopeItems = append(dataScopeItems, dataScopeItem)
		default:
			actionItems = append(actionItems, item)
		}
	}

	return actionItems, menuItems, dataScopeItems
}
