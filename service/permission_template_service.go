package service

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

type PermissionTemplateItemInput struct {
	ResourceKey string `json:"resource_key"`
	ActionKey   string `json:"action_key"`
	Allowed     bool   `json:"allowed"`
}

type PermissionTemplateUpsertRequest struct {
	ProfileName string                        `json:"profile_name"`
	ProfileType string                        `json:"profile_type"`
	Description string                        `json:"description"`
	Status      int                           `json:"status"`
	Items       []PermissionTemplateItemInput `json:"items"`
}

type PermissionTemplateDetail struct {
	Profile model.PermissionProfile      `json:"profile"`
	Items   []model.PermissionProfileItem `json:"items"`
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

	return &PermissionTemplateDetail{
		Profile: profile,
		Items:   items,
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

	if err := replacePermissionTemplateItemsTx(tx, profile.Id, req.Items, now); err != nil {
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

	if err := replacePermissionTemplateItemsTx(tx, profile.Id, req.Items, now); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return GetPermissionTemplateDetail(profile.Id)
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
	return nil
}

func normalizeTemplateStatus(status int) int {
	if status == model.CommonStatusDisabled {
		return model.CommonStatusDisabled
	}
	return model.CommonStatusEnabled
}

func replacePermissionTemplateItemsTx(tx *gorm.DB, profileId int, items []PermissionTemplateItemInput, now int64) error {
	if err := tx.Where("profile_id = ?", profileId).Delete(&model.PermissionProfileItem{}).Error; err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}

	createItems := make([]model.PermissionProfileItem, 0, len(items))
	for _, item := range items {
		createItems = append(createItems, model.PermissionProfileItem{
			ProfileId:   profileId,
			ResourceKey: item.ResourceKey,
			ActionKey:   item.ActionKey,
			Allowed:     item.Allowed,
			CreatedAtTs: now,
		})
	}
	return tx.Create(&createItems).Error
}
