export const getInviteOwnerId = (record) => {
  const inviterId = Number(record?.inviter_id) || 0;
  if (inviterId > 0) {
    return inviterId;
  }

  const parentAgentId = Number(record?.parent_agent_id) || 0;
  if (parentAgentId > 0) {
    return parentAgentId;
  }

  return 0;
};

export const getInviteOwnerName = (record) => {
  const inviterUsername = (record?.inviter_username || '').trim();
  if (inviterUsername !== '') {
    return inviterUsername;
  }

  const parentAgentUsername = (record?.parent_agent_username || '').trim();
  if (parentAgentUsername !== '') {
    return parentAgentUsername;
  }

  return '';
};
