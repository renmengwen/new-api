import test from 'node:test';
import assert from 'node:assert/strict';

import { getInviteOwnerId } from './inviteHelpers.js';
import { getInviteOwnerName } from './inviteHelpers.js';

test('getInviteOwnerId prefers inviter_id when present', () => {
  assert.equal(getInviteOwnerId({ inviter_id: 23, parent_agent_id: 9 }), 23);
});

test('getInviteOwnerId falls back to parent_agent_id for managed users', () => {
  assert.equal(getInviteOwnerId({ inviter_id: 0, parent_agent_id: 9 }), 9);
  assert.equal(getInviteOwnerId({ parent_agent_id: 9 }), 9);
});

test('getInviteOwnerId returns 0 when no inviter or parent agent exists', () => {
  assert.equal(getInviteOwnerId({ inviter_id: 0, parent_agent_id: 0 }), 0);
  assert.equal(getInviteOwnerId({}), 0);
});

test('getInviteOwnerName prefers inviter username when present', () => {
  assert.equal(
    getInviteOwnerName({
      inviter_username: 'direct_inviter',
      parent_agent_username: 'agent_parent',
    }),
    'direct_inviter',
  );
});

test('getInviteOwnerName falls back to parent agent username for managed users', () => {
  assert.equal(
    getInviteOwnerName({
      inviter_username: '',
      parent_agent_username: 'agent_parent',
    }),
    'agent_parent',
  );
  assert.equal(
    getInviteOwnerName({
      parent_agent_username: 'agent_parent',
    }),
    'agent_parent',
  );
});

test('getInviteOwnerName returns empty string when no invite owner username exists', () => {
  assert.equal(getInviteOwnerName({}), '');
  assert.equal(
    getInviteOwnerName({
      inviter_username: '',
      parent_agent_username: '',
    }),
    '',
  );
});
