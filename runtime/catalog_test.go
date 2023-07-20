package runtime_test

import (
	"context"
	"testing"

	"github.com/rilldata/rill/runtime/services/catalog"
	_ "github.com/rilldata/rill/runtime/services/catalog/artifacts/sql"
	_ "github.com/rilldata/rill/runtime/services/catalog/artifacts/yaml"
	_ "github.com/rilldata/rill/runtime/services/catalog/migrator/metricsviews"
	_ "github.com/rilldata/rill/runtime/services/catalog/migrator/models"
	_ "github.com/rilldata/rill/runtime/services/catalog/migrator/sources"
	"github.com/rilldata/rill/runtime/services/catalog/testutils"
	"github.com/rilldata/rill/runtime/testruntime"
	"github.com/stretchr/testify/require"
)

func TestCatalog(t *testing.T) {
	ctx := context.Background()
	rt, instanceID := testruntime.NewInstanceForProject(t, "ad_bids")

	cat, err := rt.NewCatalogService(ctx, instanceID)
	require.NoError(t, err)

	sourcePath := "/sources/ad_bids_source.yaml"
	modelPath := "/models/ad_bids.sql"
	metricsPath := "/dashboards/ad_bids_metrics.yaml"

	testutils.AssertTable(t, cat, "ad_bids_source", sourcePath)
	testutils.AssertTable(t, cat, "ad_bids", modelPath)

	// force update the source
	res, err := cat.Reconcile(ctx, catalog.ReconcileConfig{
		ChangedPaths: []string{sourcePath},
		ForcedPaths:  []string{sourcePath},
	})
	require.NoError(t, err)
	testutils.AssertMigration(t, res, 0, 0, 3, 0, []string{sourcePath, modelPath, metricsPath})
}

func TestCatalogAccess(t *testing.T) {
	ctx := context.Background()
	rt, instanceID := testruntime.NewInstanceForProject(t, "ad_bids")

	cat, err := rt.NewCatalogService(ctx, instanceID)
	require.NoError(t, err)

	access, err := cat.GetAccess(ctx)
	require.NoError(t, err)

	require.Equal(t, "allow", access.DefaultModelAccess)
	require.Equal(t, 2, len(access.Claims))

	require.Equal(t, "tenant_id", access.Claims[0].Name)
	require.Equal(t, "string", access.Claims[0].Type)
	require.Equal(t, "aaa", access.Claims[0].Default)
	require.Equal(t, 0, len(access.Claims[0].Options))
	require.Nil(t, access.Claims[0].Options)

	require.Equal(t, "countryCode", access.Claims[1].Name)
	require.Equal(t, "string", access.Claims[1].Type)
	require.Equal(t, "", access.Claims[1].Default)
	require.Equal(t, 2, len(access.Claims[1].Options))
	require.Equal(t, "IN", access.Claims[1].Options[0])
	require.Equal(t, "DK", access.Claims[1].Options[1])
}
