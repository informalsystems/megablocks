package xchange_test

import (
	"testing"

	keepertest "github.com/informalsystems/megablocks/app/xchange/testutil/keeper"
	"github.com/informalsystems/megablocks/app/xchange/testutil/nullify"
	xchange "github.com/informalsystems/megablocks/app/xchange/x/xchange/module"
	"github.com/informalsystems/megablocks/app/xchange/x/xchange/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.XchangeKeeper(t)
	xchange.InitGenesis(ctx, k, genesisState)
	got := xchange.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
