package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdminActionCatalogIncludesAllUserManagementActions(t *testing.T) {
	require.Equal(t, []string{
		ActionRead,
		ActionCreate,
		ActionUpdate,
		ActionUpdateStatus,
		ActionDelete,
		"reset_passkey",
		"reset_2fa",
		"manage_subscriptions",
		"manage_bindings",
	}, adminActionCatalog[ResourceUserManagement])
}

func TestAdminActionCatalogIncludesModelMonitorPermissions(t *testing.T) {
	require.Equal(t, []string{
		ActionRead,
		ActionUpdate,
		ActionTest,
	}, adminActionCatalog[ResourceModelMonitorManagement])
	require.Contains(t, adminSidebarModuleCatalog["admin"], "model-monitor")
}
