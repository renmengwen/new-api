package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAdminQuotaCostSummaryQueryDefaultsToLast7Days(t *testing.T) {
	query, err := dto.NormalizeAdminQuotaCostSummaryQuery(dto.AdminQuotaCostSummaryQuery{}, 1714320000)
	require.NoError(t, err)
	require.Equal(t, int64(1713715200), query.StartTimestamp)
	require.Equal(t, int64(1714320000), query.EndTimestamp)
	require.Equal(t, "date", query.SortBy)
	require.Equal(t, "desc", query.SortOrder)
}

func TestNormalizeAdminQuotaCostSummaryQueryRejectsRangeOver90Days(t *testing.T) {
	_, err := dto.NormalizeAdminQuotaCostSummaryQuery(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714320000 - 91*24*60*60,
		EndTimestamp:   1714320000,
	}, 1714320000)
	require.Error(t, err)
	require.Contains(t, err.Error(), "date range cannot exceed 90 days")
}

func TestNormalizeAdminQuotaCostSummaryQueryNormalizesSort(t *testing.T) {
	query, err := dto.NormalizeAdminQuotaCostSummaryQuery(dto.AdminQuotaCostSummaryQuery{
		StartTimestamp: 1714233600,
		EndTimestamp:   1714320000,
		SortBy:         "paid_usd",
		SortOrder:      "ASC",
	}, 1714320000)
	require.NoError(t, err)
	require.Equal(t, "paid_usd", query.SortBy)
	require.Equal(t, "asc", query.SortOrder)
}
