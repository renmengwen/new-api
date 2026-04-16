package service

import "github.com/QuantumNous/new-api/model"

type userInviteOwnerLookup struct {
	Id       int
	Username string
}

func PopulateUserInviteOwnerUsernames(users []*model.User) error {
	if len(users) == 0 {
		return nil
	}

	idSet := make(map[int]struct{})
	for _, user := range users {
		if user == nil {
			continue
		}
		if user.InviterId > 0 {
			idSet[user.InviterId] = struct{}{}
		}
		if user.ParentAgentId > 0 {
			idSet[user.ParentAgentId] = struct{}{}
		}
	}

	if len(idSet) == 0 {
		return nil
	}

	ids := make([]int, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	var owners []userInviteOwnerLookup
	if err := model.DB.Model(&model.User{}).Select("id, username").Where("id IN ?", ids).Find(&owners).Error; err != nil {
		return err
	}

	usernameByID := make(map[int]string, len(owners))
	for _, owner := range owners {
		usernameByID[owner.Id] = owner.Username
	}

	for _, user := range users {
		if user == nil {
			continue
		}
		user.InviterUsername = usernameByID[user.InviterId]
		user.ParentAgentUsername = usernameByID[user.ParentAgentId]
	}

	return nil
}
