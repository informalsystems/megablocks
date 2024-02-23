package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/informalsystems/megablocks/app/xchange/testutil/keeper"
	"github.com/informalsystems/megablocks/app/xchange/x/xchange/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.XchangeKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
