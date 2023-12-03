package app

import (
	// "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/nexablock/nexablock/utils"
)

// ScheduleForkUpgrade executes any necessary fork logic for based upon the current
// block height and chain ID (mainnet or testnet). It sets an upgrade plan once
// the chain reaches the pre-defined upgrade height.
//
// CONTRACT: for this logic to work properly it is required to:
//
//  1. Release a non-breaking patch version so that the chain can set the scheduled upgrade plan at upgrade-height.
//  2. Release the software defined in the upgrade-info
func (app *Nexa) ScheduleForkUpgrade(ctx sdk.Context) {
	// NOTE: there are no testnet forks for the existing versions
	if !utils.IsMainnet(ctx.ChainID()) {
		return
	}



	// handle mainnet forks with their corresponding upgrade name and info
	switch ctx.BlockHeight() {
	default:
		// No-op
		return
	}

	
}
