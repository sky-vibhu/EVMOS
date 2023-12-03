package types

import (
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// AttoNexa defines the default coin denomination used in Nexa in:
	//
	// - Staking parameters: denomination used as stake in the dPoS chain
	// - Mint parameters: denomination minted due to fee distribution rewards
	// - Governance parameters: denomination used for spam prevention in proposal deposits
	// - Crisis parameters: constant fee denomination used for spam prevention to check broken invariant
	// - EVM parameters: denomination used for running EVM state transitions in Nexa.
	AttoNexa string = "aNEXB"

	// BaseDenomUnit defines the base denomination unit for Nexa.
	// 1 nexa = 1x10^{BaseDenomUnit} aNEXB
	BaseDenomUnit = 18

	// DefaultGasPrice is default gas price for evm transactions
	DefaultGasPrice = 20
)

// PowerReduction defines the default power reduction value for staking
var PowerReduction = sdkmath.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(BaseDenomUnit), nil))

// NewNexaCoin is a utility function that returns an "aNEXB" coin with the given sdkmath.Int amount.
// The function will panic if the provided amount is negative.
func NewNexaCoin(amount sdkmath.Int) sdk.Coin {
	return sdk.NewCoin(AttoNexa, amount)
}

// NewNexaDecCoin is a utility function that returns an "aNEXB" decimal coin with the given sdkmath.Int amount.
// The function will panic if the provided amount is negative.
func NewNexaDecCoin(amount sdkmath.Int) sdk.DecCoin {
	return sdk.NewDecCoin(AttoNexa, amount)
}

// NewNexaCoinInt64 is a utility function that returns an "aNEXB" coin with the given int64 amount.
// The function will panic if the provided amount is negative.
func NewNexaCoinInt64(amount int64) sdk.Coin {
	return sdk.NewInt64Coin(AttoNexa, amount)
}
