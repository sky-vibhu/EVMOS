
package ics20_test

import (
	"testing"
	"time"

	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	nexaapp "github.com/nexablock/nexablock/app"
	nexaibc "github.com/nexablock/nexablock/ibc/testing"
	"github.com/nexablock/nexablock/precompiles/ics20"
	"github.com/nexablock/nexablock/x/evm/statedb"
	evmtypes "github.com/nexablock/nexablock/x/evm/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
)

var s *PrecompileTestSuite

type PrecompileTestSuite struct {
	suite.Suite

	ctx           sdk.Context
	app           *nexaapp.Nexa
	address       common.Address
	differentAddr common.Address
	validators    []stakingtypes.Validator
	valSet        *tmtypes.ValidatorSet
	ethSigner     ethtypes.Signer
	privKey       cryptotypes.PrivKey
	signer        keyring.Signer
	bondDenom     string

	precompile *ics20.Precompile
	stateDB    *statedb.StateDB

	coordinator    *ibctesting.Coordinator
	chainA         *ibctesting.TestChain
	chainB         *ibctesting.TestChain
	transferPath   *nexaibc.Path
	queryClientEVM evmtypes.QueryClient

	defaultExpirationDuration time.Time

	suiteIBCTesting bool
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(PrecompileTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "ICS20 Precompile Suite")
}

func (s *PrecompileTestSuite) SetupTest() {
	s.DoSetupTest()
}
