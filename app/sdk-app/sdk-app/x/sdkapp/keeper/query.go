package keeper

import (
	"github.com/megablocks/sdk-app/x/sdkapp/types"
)

var _ types.QueryServer = Keeper{}
