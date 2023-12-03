package network

import (
	"encoding/json"
	"math"

	"github.com/nexablock/nexablock/app"
	"github.com/nexablock/nexablock/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	evmtypes "github.com/nexablock/nexablock/x/evm/types"
	feemarkettypes "github.com/nexablock/nexablock/x/feemarket/types"
	
)

// Network is the interface that wraps the methods to interact with integration test network.
//
// It was designed to avoid users to access module's keepers directly and force integration tests
// to be closer to the real user's behavior.
type Network interface {
	GetContext() sdktypes.Context
	GetChainID() string
	GetDenom() string
	GetValidators() []stakingtypes.Validator

	NextBlock() error

	GetEvmClient() evmtypes.QueryClient
	
	GetBankClient() banktypes.QueryClient
	GetFeeMarketClient() feemarkettypes.QueryClient
	GetAuthClient() authtypes.QueryClient

	// Because to update the module params on a conventional manner governance
	// would be require, we should provide an easier way to update the params
	
	UpdateEvmParams(params evmtypes.Params) error

	BroadcastTxSync(txBytes []byte) (abcitypes.ResponseDeliverTx, error)
	Simulate(txBytes []byte) (*txtypes.SimulateResponse, error)
}

var _ Network = (*IntegrationNetwork)(nil)

// IntegrationNetwork is the implementation of the Network interface for integration tests.
type IntegrationNetwork struct {
	cfg        Config
	ctx        sdktypes.Context
	app        *app.Nexa
	validators []stakingtypes.Validator
}

// New configures and initializes a new integration Network instance with
// the given configuration options. If no configuration options are provided
// it uses the default configuration.
//
// It panics if an error occurs.
func New(opts ...ConfigOption) *IntegrationNetwork {
	cfg := DefaultConfig()
	// Modify the default config with the given options
	for _, opt := range opts {
		opt(&cfg)
	}

	ctx := sdktypes.Context{}
	network := &IntegrationNetwork{
		cfg:        cfg,
		ctx:        ctx,
		validators: []stakingtypes.Validator{},
	}

	err := network.configureAndInitChain()
	if err != nil {
		panic(err)
	}
	return network
}

var (
	// bondedAmt is the amount of tokens that each validator will have initially bonded
	bondedAmt = sdktypes.TokensFromConsensusPower(1, types.PowerReduction)
	// PrefundedAccountInitialBalance is the amount of tokens that each prefunded account has at genesis
	PrefundedAccountInitialBalance = sdktypes.NewInt(int64(math.Pow10(18) * 4))
)

// configureAndInitChain initializes the network with the given configuration.
// It creates the genesis state and starts the network.
func (n *IntegrationNetwork) configureAndInitChain() error {
	// Create funded accounts based on the config and
	// create genesis accounts
	coin := sdktypes.NewCoin(n.cfg.denom, PrefundedAccountInitialBalance)
	genAccounts := createGenesisAccounts(n.cfg.preFundedAccounts)
	fundedAccountBalances := createBalances(n.cfg.preFundedAccounts, coin)

	// Create validator set with the amount of validators specified in the config
	// with the default power of 1.
	valSet := createValidatorSet(n.cfg.amountOfValidators)
	totalBonded := bondedAmt.Mul(sdktypes.NewInt(int64(n.cfg.amountOfValidators)))

	// Build staking type validators and delegations
	validators, err := createStakingValidators(valSet.Validators, bondedAmt)
	if err != nil {
		return err
	}

	fundedAccountBalances = addBondedModuleAccountToFundedBalances(fundedAccountBalances, sdktypes.NewCoin(n.cfg.denom, totalBonded))

	delegations := createDelegations(valSet.Validators, genAccounts[0].GetAddress())

	// Create a new NexaApp with the following params
	nexaApp := createNexaApp(n.cfg.chainID)

	// Configure Genesis state
	genesisState := app.NewDefaultGenesisState()

	genesisState = setAuthGenesisState(nexaApp, genesisState, genAccounts)

	stakingParams := StakingCustomGenesisState{
		denom:       n.cfg.denom,
		validators:  validators,
		delegations: delegations,
	}
	genesisState = setStakingGenesisState(nexaApp, genesisState, stakingParams)

	

	totalSupply := calculateTotalSupply(fundedAccountBalances)
	bankParams := BankCustomGenesisState{
		totalSupply: totalSupply,
		balances:    fundedAccountBalances,
	}
	genesisState = setBankGenesisState(nexaApp, genesisState, bankParams)

	// Init chain
	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	if err != nil {
		return err
	}

	nexaApp.InitChain(
		abcitypes.RequestInitChain{
			ChainId:         n.cfg.chainID,
			Validators:      []abcitypes.ValidatorUpdate{},
			ConsensusParams: app.DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)
	// Commit genesis changes
	nexaApp.Commit()

	header := tmproto.Header{
		ChainID:            n.cfg.chainID,
		Height:             nexaApp.LastBlockHeight() + 1,
		AppHash:            nexaApp.LastCommitID().Hash,
		ValidatorsHash:     valSet.Hash(),
		NextValidatorsHash: valSet.Hash(),
		ProposerAddress:    valSet.Proposer.Address,
	}
	nexaApp.BeginBlock(abcitypes.RequestBeginBlock{Header: header})

	// Set networks global parameters
	n.app = nexaApp
	// TODO - this might not be the best way to initilize the context
	n.ctx = nexaApp.BaseApp.NewContext(false, header)
	n.validators = validators
	return nil
}

// GetContext returns the network's context
func (n *IntegrationNetwork) GetContext() sdktypes.Context {
	return n.ctx
}

// GetChainID returns the network's chainID
func (n *IntegrationNetwork) GetChainID() string {
	return n.cfg.chainID
}

// GetDenom returns the network's denom
func (n *IntegrationNetwork) GetDenom() string {
	return n.cfg.denom
}

// GetValidators returns the network's validators
func (n *IntegrationNetwork) GetValidators() []stakingtypes.Validator {
	return n.validators
}

// BroadcastTxSync broadcasts the given txBytes to the network and returns the response.
// TODO - this should be change to gRPC
func (n *IntegrationNetwork) BroadcastTxSync(txBytes []byte) (abcitypes.ResponseDeliverTx, error) {
	req := abcitypes.RequestDeliverTx{Tx: txBytes}
	return n.app.BaseApp.DeliverTx(req), nil
}

// Simulate simulates the given txBytes to the network and returns the simulated response.
// TODO - this should be change to gRPC
func (n *IntegrationNetwork) Simulate(txBytes []byte) (*txtypes.SimulateResponse, error) {
	gas, result, err := n.app.BaseApp.Simulate(txBytes)
	if err != nil {
		return nil, err
	}
	return &txtypes.SimulateResponse{
		GasInfo: &gas,
		Result:  result,
	}, nil
}
