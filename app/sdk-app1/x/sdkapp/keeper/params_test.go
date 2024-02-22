package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/informalsystems/megablocks/app/sdk-app1/testutil/keeper"
	"github.com/informalsystems/megablocks/app/sdk-app1/x/sdkapp/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.SdkappKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
