package sdkapp_test

import (
	"testing"

	keepertest "github.com/megablocks/sdk-app/testutil/keeper"
	"github.com/megablocks/sdk-app/testutil/nullify"
	sdkapp "github.com/megablocks/sdk-app/x/sdkapp/module"
	"github.com/megablocks/sdk-app/x/sdkapp/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.SdkappKeeper(t)
	sdkapp.InitGenesis(ctx, k, genesisState)
	got := sdkapp.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
