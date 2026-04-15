package service

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestBuildExcelFileWritesHeadersAndRows(t *testing.T) {
	fileName, content, err := BuildExcelFile(ExcelFileSpec{
		FileNamePrefix: "审计日志",
		SheetName:      "审计日志",
		Headers:        []string{"ID", "操作人", "时间"},
		Rows: [][]string{
			{"101", "alice [ID:1]", "2026-04-16 12:00:00"},
			{"102", "bob [ID:2]", "2026-04-16 12:05:00"},
		},
	})
	require.NoError(t, err)
	require.Regexp(t, `^审计日志_\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}\.xlsx$`, fileName)

	workbook, err := excelize.OpenReader(bytes.NewReader(content))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, workbook.Close())
	})

	require.Equal(t, "ID", mustCell(t, workbook, "审计日志", "A1"))
	require.Equal(t, "操作人", mustCell(t, workbook, "审计日志", "B1"))
	require.Equal(t, "时间", mustCell(t, workbook, "审计日志", "C1"))
	require.Equal(t, "101", mustCell(t, workbook, "审计日志", "A2"))
	require.Equal(t, "alice [ID:1]", mustCell(t, workbook, "审计日志", "B2"))
	require.Equal(t, "2026-04-16 12:00:00", mustCell(t, workbook, "审计日志", "C2"))
	require.Equal(t, "102", mustCell(t, workbook, "审计日志", "A3"))
	require.Equal(t, "bob [ID:2]", mustCell(t, workbook, "审计日志", "B3"))
	require.Equal(t, "2026-04-16 12:05:00", mustCell(t, workbook, "审计日志", "C3"))
}

func mustCell(t *testing.T, workbook *excelize.File, sheetName string, cell string) string {
	t.Helper()

	value, err := workbook.GetCellValue(sheetName, cell)
	require.NoError(t, err)

	return value
}
