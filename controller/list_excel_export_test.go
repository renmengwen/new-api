package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type auditExportFixture struct {
	LatestMatching   model.AdminAuditLog
	ModuleMismatch   model.AdminAuditLog
	OperatorMismatch model.AdminAuditLog
}

type quotaLedgerExportFixture struct {
	LatestMatching    model.QuotaLedger
	EntryTypeMismatch model.QuotaLedger
	UserMismatch      model.QuotaLedger
	OperatorMismatch  model.QuotaLedger
}

func TestExportAdminAuditLogsUsesFiltersAndCap(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedAuditLogs(t, db, 2050, "quota", 9001)

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/admin/audit-logs/export", map[string]any{
		"action_module":    "quota",
		"operator_user_id": 9001,
		"limit":            2000,
	})

	ExportAdminAuditLogs(ctx)

	require.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", recorder.Header().Get("Content-Type"))
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "审计日志_")

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("审计日志")
	require.NoError(t, err)
	require.Len(t, rows, 2001)
	require.Equal(t, "操作人", rows[0][1])

	dataRows := rows[1:]
	exportedIDs := sheetColumnValues(dataRows, 0)
	require.Equal(t, strconv.Itoa(fixture.LatestMatching.Id), dataRows[0][0])
	require.Contains(t, dataRows[0][1], "[ID:9001]")
	require.Equal(t, "quota", dataRows[0][2])
	require.Equal(t, "adjust", dataRows[0][3])
	require.NotContains(t, exportedIDs, strconv.Itoa(fixture.ModuleMismatch.Id))
	require.NotContains(t, exportedIDs, strconv.Itoa(fixture.OperatorMismatch.Id))
	requireStrictlyDescendingIDs(t, exportedIDs)

	for _, row := range dataRows {
		require.Equal(t, "quota", row[2])
		require.Contains(t, row[1], "[ID:9001]")
	}
}

func TestExportQuotaLedgerUsesEntryTypeFilterAndCap(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	fixture := seedQuotaLedgerRows(t, db, 2088, model.LedgerEntryAdjust)

	ctx, recorder := newSettingAuditContext(t, http.MethodPost, "/api/admin/quota/ledger/export", map[string]any{
		"user_id":          2001,
		"operator_user_id": 9001,
		"entry_type":       model.LedgerEntryAdjust,
		"limit":            2000,
	})

	ExportQuotaLedger(ctx)

	workbook := openWorkbookBytes(t, recorder.Body.Bytes())
	rows, err := workbook.GetRows("额度流水")
	require.NoError(t, err)
	require.Len(t, rows, 2001)
	require.Equal(t, "类型", rows[0][3])

	dataRows := rows[1:]
	exportedIDs := sheetColumnValues(dataRows, 0)
	require.Equal(t, strconv.Itoa(fixture.LatestMatching.Id), dataRows[0][0])
	require.Equal(t, "quota_user", dataRows[0][1])
	require.Contains(t, dataRows[0][2], "[ID:9001]")
	require.Equal(t, model.LedgerEntryAdjust, dataRows[0][3])
	require.NotContains(t, exportedIDs, strconv.Itoa(fixture.EntryTypeMismatch.Id))
	require.NotContains(t, exportedIDs, strconv.Itoa(fixture.UserMismatch.Id))
	require.NotContains(t, exportedIDs, strconv.Itoa(fixture.OperatorMismatch.Id))
	requireStrictlyDescendingIDs(t, exportedIDs)

	for _, row := range dataRows {
		require.Equal(t, "quota_user", row[1])
		require.Contains(t, row[2], "[ID:9001]")
		require.Equal(t, model.LedgerEntryAdjust, row[3])
	}
}

func TestExportAdminAuditLogsServiceHelperCapsLimit(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	seedAuditLogs(t, db, 2050, "quota", 9001)

	items, _, err := service.ListAdminAuditLogsForExport("quota", 9001, 5000)
	require.NoError(t, err)
	require.Len(t, items, 2000)
	require.True(t, items[0].Id > items[len(items)-1].Id)

	items, _, err = service.ListAdminAuditLogsForExport("quota", 9001, 123)
	require.NoError(t, err)
	require.Len(t, items, 123)
}

func TestExportQuotaLedgerServiceHelperCapsLimit(t *testing.T) {
	db := setupListExcelExportTestDB(t)
	seedQuotaLedgerRows(t, db, 2088, model.LedgerEntryAdjust)

	items, _, err := service.ListQuotaLedgerForExport(9001, common.RoleRootUser, 2001, 9001, model.LedgerEntryAdjust, 5000)
	require.NoError(t, err)
	require.Len(t, items, 2000)
	require.True(t, items[0].Id > items[len(items)-1].Id)

	items, _, err = service.ListQuotaLedgerForExport(9001, common.RoleRootUser, 2001, 9001, model.LedgerEntryAdjust, 321)
	require.NoError(t, err)
	require.Len(t, items, 321)
}

func setupListExcelExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := setupSettingAuditTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.QuotaAccount{},
		&model.QuotaLedger{},
	))
	return db
}

func openWorkbookBytes(t *testing.T, content []byte) *excelize.File {
	t.Helper()

	workbook, err := excelize.OpenReader(bytes.NewReader(content))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, workbook.Close())
	})
	return workbook
}

func seedAuditLogs(t *testing.T, db *gorm.DB, total int, actionModule string, operatorUserID int) auditExportFixture {
	t.Helper()

	seedListExportUsers(t, db,
		testListExportUser(operatorUserID, "root_operator", "Root Operator", common.RoleRootUser, model.UserTypeRoot),
		testListExportUser(1001, "target_user", "Target User", common.RoleCommonUser, model.UserTypeEndUser),
		testListExportUser(9002, "other_operator", "Other Operator", common.RoleAdminUser, model.UserTypeAdmin),
	)

	logs := make([]model.AdminAuditLog, 0, total+2)
	for i := 0; i < total; i++ {
		logs = append(logs, model.AdminAuditLog{
			OperatorUserId:   operatorUserID,
			OperatorUserType: model.UserTypeRoot,
			ActionModule:     actionModule,
			ActionType:       "adjust",
			ActionDesc:       "quota_adjust",
			TargetType:       "user",
			TargetId:         1001,
			Ip:               "203.0.113.9",
			CreatedAtTs:      int64(1710000000 + i),
		})
	}
	logs = append(logs,
		model.AdminAuditLog{
			OperatorUserId:   operatorUserID,
			OperatorUserType: model.UserTypeRoot,
			ActionModule:     "setting_misc",
			ActionType:       "module_mismatch",
			ActionDesc:       "other_module",
			TargetType:       "user",
			TargetId:         1001,
			Ip:               "198.51.100.10",
			CreatedAtTs:      1810000001,
		},
		model.AdminAuditLog{
			OperatorUserId:   9002,
			OperatorUserType: model.UserTypeAdmin,
			ActionModule:     actionModule,
			ActionType:       "operator_mismatch",
			ActionDesc:       "other_operator",
			TargetType:       "user",
			TargetId:         1001,
			Ip:               "198.51.100.11",
			CreatedAtTs:      1810000002,
		},
	)

	require.NoError(t, db.CreateInBatches(logs, 200).Error)
	return auditExportFixture{
		LatestMatching:   logs[total-1],
		ModuleMismatch:   logs[total],
		OperatorMismatch: logs[total+1],
	}
}

func seedQuotaLedgerRows(t *testing.T, db *gorm.DB, total int, entryType string) quotaLedgerExportFixture {
	t.Helper()

	seedListExportUsers(t, db,
		testListExportUser(9001, "root_operator", "Root Operator", common.RoleRootUser, model.UserTypeRoot),
		testListExportUser(9002, "other_operator", "Other Operator", common.RoleAdminUser, model.UserTypeAdmin),
		testListExportUser(2001, "quota_user", "Quota User", common.RoleCommonUser, model.UserTypeEndUser),
		testListExportUser(2002, "other_user", "Other User", common.RoleCommonUser, model.UserTypeEndUser),
	)

	account := &model.QuotaAccount{
		OwnerType:        model.QuotaOwnerTypeUser,
		OwnerId:          2001,
		Balance:          500000,
		TotalRecharged:   500000,
		TotalAdjustedIn:  0,
		TotalAdjustedOut: 0,
		Status:           model.CommonStatusEnabled,
		CreatedAtTs:      1710000000,
		UpdatedAtTs:      1710000000,
	}
	require.NoError(t, db.Create(account).Error)

	otherAccount := &model.QuotaAccount{
		OwnerType:        model.QuotaOwnerTypeUser,
		OwnerId:          2002,
		Balance:          1000,
		TotalRecharged:   1000,
		TotalAdjustedIn:  0,
		TotalAdjustedOut: 0,
		Status:           model.CommonStatusEnabled,
		CreatedAtTs:      1710000000,
		UpdatedAtTs:      1710000000,
	}
	require.NoError(t, db.Create(otherAccount).Error)

	ledgers := make([]model.QuotaLedger, 0, total+3)
	for i := 0; i < total; i++ {
		before := 100000 + i
		ledgers = append(ledgers, model.QuotaLedger{
			BizNo:            fmt.Sprintf("ql_export_%d", i),
			AccountId:        account.Id,
			TransferOrderId:  0,
			EntryType:        entryType,
			Direction:        model.LedgerDirectionIn,
			Amount:           1,
			BalanceBefore:    before,
			BalanceAfter:     before + 1,
			SourceType:       "admin_quota_adjust",
			SourceId:         2001,
			OperatorUserId:   9001,
			OperatorUserType: model.UserTypeRoot,
			Reason:           "adjust",
			Remark:           "",
			CreatedAtTs:      int64(1710000000 + i),
		})
	}
	ledgers = append(ledgers,
		model.QuotaLedger{
			BizNo:            "ql_export_other_type",
			AccountId:        account.Id,
			TransferOrderId:  0,
			EntryType:        model.LedgerEntryRecharge,
			Direction:        model.LedgerDirectionIn,
			Amount:           10,
			BalanceBefore:    1,
			BalanceAfter:     11,
			SourceType:       "topup",
			SourceId:         2001,
			OperatorUserId:   9001,
			OperatorUserType: model.UserTypeRoot,
			Reason:           "wrong_entry_type",
			CreatedAtTs:      1810000001,
		},
		model.QuotaLedger{
			BizNo:            "ql_export_other_account",
			AccountId:        otherAccount.Id,
			TransferOrderId:  0,
			EntryType:        entryType,
			Direction:        model.LedgerDirectionIn,
			Amount:           5,
			BalanceBefore:    10,
			BalanceAfter:     15,
			SourceType:       "admin_quota_adjust",
			SourceId:         2002,
			OperatorUserId:   9001,
			OperatorUserType: model.UserTypeRoot,
			Reason:           "wrong_user",
			CreatedAtTs:      1810000002,
		},
		model.QuotaLedger{
			BizNo:            "ql_export_other_operator",
			AccountId:        account.Id,
			TransferOrderId:  0,
			EntryType:        entryType,
			Direction:        model.LedgerDirectionIn,
			Amount:           7,
			BalanceBefore:    20,
			BalanceAfter:     27,
			SourceType:       "admin_quota_adjust",
			SourceId:         2001,
			OperatorUserId:   9002,
			OperatorUserType: model.UserTypeAdmin,
			Reason:           "wrong_operator",
			CreatedAtTs:      1810000003,
		},
	)

	require.NoError(t, db.CreateInBatches(ledgers, 200).Error)
	return quotaLedgerExportFixture{
		LatestMatching:    ledgers[total-1],
		EntryTypeMismatch: ledgers[total],
		UserMismatch:      ledgers[total+1],
		OperatorMismatch:  ledgers[total+2],
	}
}

func seedListExportUsers(t *testing.T, db *gorm.DB, users ...model.User) {
	t.Helper()

	require.NoError(t, db.Create(&users).Error)
}

func testListExportUser(id int, username string, displayName string, role int, userType string) model.User {
	return model.User{
		Id:          id,
		Username:    username,
		Password:    "hashed-password",
		DisplayName: displayName,
		Role:        role,
		Status:      common.UserStatusEnabled,
		UserType:    userType,
		Group:       "default",
		AffCode:     fmt.Sprintf("aff_%d", id),
		Quota:       0,
		Email:       fmt.Sprintf("%s@example.com", username),
	}
}

func sheetColumnValues(rows [][]string, column int) []string {
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) <= column {
			values = append(values, "")
			continue
		}
		values = append(values, row[column])
	}
	return values
}

func requireStrictlyDescendingIDs(t *testing.T, ids []string) {
	t.Helper()

	require.NotEmpty(t, ids)

	previousID, err := strconv.Atoi(ids[0])
	require.NoError(t, err)

	for _, rawID := range ids[1:] {
		currentID, err := strconv.Atoi(rawID)
		require.NoError(t, err)
		require.Less(t, currentID, previousID)
		previousID = currentID
	}
}
