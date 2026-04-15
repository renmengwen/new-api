# Wallet Ledger Hardening Design

Date: 2026-04-14

## Goal

Turn `quota_ledgers` into a strict wallet quota ledger for user balance changes.

After this stage:

- Every wallet quota balance change must create a ledger entry.
- `quota_accounts.balance` must be the authoritative wallet balance.
- Drift repair must be explicit and traceable.
- The existing quota ledger page can be treated as a strict wallet ledger view.

This stage does not attempt to unify wallet quota, subscription quota, and cash payments into one financial ledger.

## Current Problems

The current ledger covers many core wallet flows, but it is not yet complete or strict.

- Some balance-changing paths still bypass `quota_ledgers`.
  - Redemption code recharge updates `users.quota` directly and only writes a usage log.
- Some account creation paths initialize `quota_accounts` without writing an opening ledger entry.
  - Admin-created end users
  - Agent users
  - Admin manager users
  - Bootstrap root user
- Some older wallet flows silently overwrite `quota_accounts.balance` to match `users.quota` without creating a reconciliation entry.
- Legacy direct balance mutation helpers still exist and can bypass the ledger model.
- Subscription quota is tracked separately and is not part of `quota_ledgers`.

Because of these gaps, the page is currently closer to "mostly useful wallet history" than a strict auditable ledger.

## Scope

Stage 1 is limited to wallet quota.

Included:

- `quota_ledgers`
- `quota_accounts`
- Wallet quota creation, recharge, consume, refund, reward, commission, adjust, and reconcile flows
- Wallet-related admin and agent operations

Excluded:

- Subscription quota ledgering
- Cash accounting, payment currency, and settlement reconciliation
- Merging the quota ledger page into the usage logs page

## Design

### 1. Ledger Domain Definition

`quota_ledgers` becomes the strict ledger for wallet quota only.

Supported ledger semantics after this stage:

- `opening`: initial wallet balance snapshot at account creation
- `recharge`: wallet quota recharge
- `consume`: wallet quota deduction
- `refund`: wallet quota refund
- `reward`: system reward into wallet quota
- `commission`: commission or affiliate quota transferred into wallet quota
- `adjust`: manual operator adjustment
- `reconcile`: explicit drift repair between persisted balance sources

Implementation note:

- `reconcile` does not need a new `entry_type`; it can remain `adjust` with `source_type=quota_reconcile` if we want minimal schema impact.
- `opening` should be a new `entry_type` string constant because it improves readability and reporting.

### 2. Single Source of Truth

`quota_accounts.balance` becomes the authoritative wallet balance.

Rules:

- Business flows must compute and persist wallet balance through `quota_accounts`.
- `users.quota` remains as a compatibility field and must be updated in the same transaction after the ledger write.
- No business flow may update `users.quota` without also producing a ledger entry and updating `quota_accounts`.

This preserves backward compatibility while giving the system one ledger-backed balance source.

### 3. Mandatory Ledger Coverage

The following flows must always generate ledger entries.

#### 3.1 Account opening

When a wallet account is initialized with a non-zero initial quota, write an opening ledger entry:

- `entry_type=opening`
- `direction=in`
- `amount=initial_balance`
- `balance_before=0`
- `balance_after=initial_balance`
- `source_type` reflects the creation path
  - `user_register`
  - `admin_user_create`
  - `agent_create`
  - `admin_manager_create`
  - `root_bootstrap`

This applies to:

- Standard user registration
- OAuth-created users
- Admin-created managed users
- New agent users
- New admin manager users
- Bootstrap root account

#### 3.2 Redemption recharge

Redemption code recharge must stop directly mutating `users.quota`.

It should instead create a wallet recharge ledger entry:

- `entry_type=recharge`
- `source_type=redemption_recharge`
- `source_id=redemption.id`
- `reason=redemption`

The redemption record status update and wallet ledger write must remain in one transaction.

#### 3.3 Existing wallet flows

Existing wallet flows that already write ledger entries remain on the ledger path:

- online top-up recharge
- request consume/refund
- task consume/refund
- Midjourney refund
- check-in reward
- invitee reward
- affiliate transfer into wallet quota
- admin or agent manual adjustment

The implementation work here is verification and refactor safety, not a semantic redesign.

### 4. Explicit Reconciliation Only

Silent balance alignment must be removed.

Current behavior in some older paths:

- if `quota_accounts.balance != users.quota`
- overwrite `quota_accounts.balance = users.quota`
- continue processing

This is not acceptable for a strict ledger because it destroys traceability.

Required behavior:

- If drift is detected inside ledger-aware wallet flows, write an explicit reconciliation entry before applying the requested business change.
- Reconciliation should preserve:
  - before balance
  - after balance
  - operator context if any
  - `source_type=quota_reconcile`
  - `reason=sync_with_user_quota`

The newer reconciliation implementation in `service/quota_service.go` should become the shared pattern across wallet flows.

### 5. Remove Bypass Helpers from Business Use

Legacy balance mutation helpers should no longer be used by business flows:

- `IncreaseUserQuota`
- `DecreaseUserQuota`
- `DeltaUpdateUserQuota`
- batch updates that indirectly call the same direct mutation path

Allowed end state:

- The helpers may remain temporarily for compatibility, but no active wallet business flow may depend on them.
- Add comments marking them as legacy and non-ledger-safe if they cannot be removed immediately.

### 6. UI Semantics

The current page can remain visually similar in Stage 1, but its product meaning must be narrowed.

Recommended naming:

- page title: `Wallet Quota Ledger`
- helper text: this page shows wallet quota balance changes only

This avoids implying that the page already covers subscription quota or cash settlement.

## Implementation Outline

### Backend

1. Add the new `opening` ledger entry type constant.
2. Add a shared helper for creating opening ledger entries.
3. Update account creation paths to write opening entries when initial quota is non-zero.
4. Refactor redemption recharge to write wallet ledger entries transactionally.
5. Replace silent balance alignment in older reward and recharge paths with explicit reconciliation entries.
6. Audit code paths for direct `users.quota` mutations and route active wallet flows through ledger-aware services.

### Frontend

1. Update quota ledger entry type labels to include `opening`.
2. Adjust page copy from generic "quota ledger" wording toward wallet-ledger wording if approved.
3. Keep the existing list structure unless additional filtering changes are requested later.

## Validation

Add or update targeted tests for:

- admin-created user with initial quota produces an opening ledger entry
- agent creation with initial quota produces an opening ledger entry
- admin manager creation with initial quota produces an opening ledger entry
- root bootstrap creates an opening ledger entry
- redemption recharge creates a recharge ledger entry and updates account balance correctly
- reconciliation is explicit when account and user quota drift exists
- no covered wallet flow changes balance without a matching ledger entry

Manual validation:

- create a user through each supported creation path
- redeem a code
- recharge online
- perform admin and agent adjustments
- verify the ledger page shows complete wallet history and balance progression

## Non-Goals

- No subscription ledger unification in this stage
- No cash bookkeeping redesign
- No change to usage logs semantics in this stage
- No removal of compatibility fields such as `users.quota` in this stage

## Risks

- There are legacy paths that still assume `users.quota` is the primary balance field.
- Introducing `opening` changes filter and display logic on the quota ledger page.
- Some historical records will remain incomplete; this stage improves correctness for new writes, not retroactive perfection.

## Success Criteria

The stage is complete when all of the following are true:

- Every wallet quota balance mutation goes through a ledger-producing path.
- `quota_accounts.balance` and the latest wallet ledger `balance_after` are consistent.
- Drift repair is always explicit and queryable.
- The quota ledger page can be trusted as the strict history of wallet quota changes going forward.
