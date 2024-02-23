package keeper

import (
	"github.com/informalsystems/megablocks/app/xchange/x/xchange/types"
)

var _ types.QueryServer = Keeper{}
