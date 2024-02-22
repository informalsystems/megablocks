package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/megablocks/sdk-app/testutil/keeper"
	"github.com/megablocks/sdk-app/x/sdkapp/types"
)

func TestGetParams(t *testing.T) {
	k, ctx := keepertest.SdkappKeeper(t)
	params := types.DefaultParams()

	require.NoError(t, k.SetParams(ctx, params))
	require.EqualValues(t, params, k.GetParams(ctx))
}
