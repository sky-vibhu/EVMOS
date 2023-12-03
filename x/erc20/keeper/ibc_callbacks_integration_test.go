package keeper_test

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/ethereum/go-ethereum/common"
	. "github.com/onsi/ginkgo/v2"
	"github.com/nexablock/nexablock/app"
	"github.com/nexablock/nexablock/contracts"
	ibctesting "github.com/nexablock/nexablock/ibc/testing"
	teststypes "github.com/nexablock/nexablock/types/tests"
	"github.com/nexablock/nexablock/utils"
	"github.com/nexablock/nexablock/x/erc20/types"
)

var _ = Describe("Convert receiving IBC to Erc20", Ordered, func() {
	var (
		sender, receiver string
		receiverAcc      sdk.AccAddress
		senderAcc        sdk.AccAddress
		amount           int64 = 10
		pair             *types.TokenPair
		erc20Denomtrace  transfertypes.DenomTrace
	)

	// Metadata to register OSMO with a Token Pair for testing
	osmoMeta := banktypes.Metadata{
		Description: "IBC Coin for IBC Osmosis Chain",
		Base:        teststypes.UosmoIbcdenom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    teststypes.UosmoDenomtrace.BaseDenom,
				Exponent: 0,
			},
		},
		Name:    teststypes.UosmoIbcdenom,
		Symbol:  erc20Symbol,
		Display: teststypes.UosmoDenomtrace.BaseDenom,
	}

	nexaMeta := banktypes.Metadata{
		Description: "Base Denom for Nexa Chain",
		Base:        utils.BaseDenom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    teststypes.aNEXBDenomtrace.BaseDenom,
				Exponent: 0,
			},
		},
		Name:    utils.BaseDenom,
		Symbol:  erc20Symbol,
		Display: teststypes.aNEXBDenomtrace.BaseDenom,
	}

	BeforeEach(func() {
		s.suiteIBCTesting = true
		s.SetupTest()
		s.suiteIBCTesting = false
	})

	Describe("disabled params", func() {
		BeforeEach(func() {
			erc20params := types.DefaultParams()
			erc20params.EnableErc20 = false
			err := s.app.Erc20Keeper.SetParams(s.NexaChain.GetContext(), erc20params)
			s.Require().NoError(err)

			sender = s.IBCOsmosisChain.SenderAccount.GetAddress().String()
			receiver = s.NexaChain.SenderAccount.GetAddress().String()
			receiverAcc = sdk.MustAccAddressFromBech32(receiver)
		})
		It("should transfer and not convert to erc20", func() {
			// register the pair to check that it was not converted to ERC-20
			pair, err := s.app.Erc20Keeper.RegisterCoin(s.NexaChain.GetContext(), osmoMeta)
			s.Require().NoError(err)

			// check balance before transfer is 0
			ibcOsmoBalanceBefore := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(int64(0), ibcOsmoBalanceBefore.Amount.Int64())

			s.SendAndReceiveMessage(s.pathOsmosisNexa, s.IBCOsmosisChain, "uosmo", amount, sender, receiver, 1, "")

			// check balance after transfer
			ibcOsmoBalanceAfter := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(amount, ibcOsmoBalanceAfter.Amount.Int64())

			// check ERC20 balance - should be zero (no conversion)
			balanceERC20TokenAfter := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
			s.Require().Equal(int64(0), balanceERC20TokenAfter.Int64())
		})
	})
	Describe("enabled params and registered uosmo", func() {
		BeforeEach(func() {
			erc20params := types.DefaultParams()
			erc20params.EnableErc20 = true
			err := s.app.Erc20Keeper.SetParams(s.NexaChain.GetContext(), erc20params)
			s.Require().NoError(err)

			sender = s.IBCOsmosisChain.SenderAccount.GetAddress().String()
			receiver = s.NexaChain.SenderAccount.GetAddress().String()
			senderAcc = sdk.MustAccAddressFromBech32(sender)
			receiverAcc = sdk.MustAccAddressFromBech32(receiver)

			// Register uosmo pair
			pair, err = s.app.Erc20Keeper.RegisterCoin(s.NexaChain.GetContext(), osmoMeta)
			s.Require().NoError(err)
		})
		It("should transfer and convert uosmo to tokens", func() {
			// Check receiver's balance for IBC and ERC-20 before transfer. Should be zero
			balanceTokenBefore := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
			s.Require().Equal(int64(0), balanceTokenBefore.Int64())

			ibcOsmoBalanceBefore := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(int64(0), ibcOsmoBalanceBefore.Amount.Int64())

			s.NexaChain.Coordinator.CommitBlock()
			// Send coins
			s.SendAndReceiveMessage(s.pathOsmosisNexa, s.IBCOsmosisChain, "uosmo", amount, sender, receiver, 1, "")

			// Check ERC20 balances
			balanceTokenAfter := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
			s.Require().Equal(amount, balanceTokenAfter.Int64())

			// Check IBC uosmo coin balance - should be zero
			ibcOsmoBalanceAfter := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(int64(0), ibcOsmoBalanceAfter.Amount.Int64())
		})
		It("should transfer and not convert unregistered coin (uatom)", func() {
			sender = s.IBCCosmosChain.SenderAccount.GetAddress().String()

			// check balance before transfer is 0
			ibcAtomBalanceBefore := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, teststypes.UatomIbcdenom)
			s.Require().Equal(int64(0), ibcAtomBalanceBefore.Amount.Int64())

			s.NexaChain.Coordinator.CommitBlock()
			s.SendAndReceiveMessage(s.pathCosmosNexa, s.IBCCosmosChain, "uatom", amount, sender, receiver, 1, "")

			// check balance after transfer
			ibcAtomBalanceAfter := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, teststypes.UatomIbcdenom)
			s.Require().Equal(amount, ibcAtomBalanceAfter.Amount.Int64())
		})
		It("should transfer and not convert aNEXB", func() {
			// Register 'aNEXB' coin in ERC-20 keeper to validate it is not converting the coins when receiving 'aNEXB' thru IBC
			pair, err := s.app.Erc20Keeper.RegisterCoin(s.NexaChain.GetContext(), nexaMeta)
			s.Require().NoError(err)

			aNEXBInitialBalance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, utils.BaseDenom)

			// 1. Send aNEXB from Nexa to Osmosis
			s.SendAndReceiveMessage(s.pathOsmosisNexa, s.NexaChain, utils.BaseDenom, amount, receiver, sender, 1, "")

			aNEXBAfterBalance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, utils.BaseDenom)
			s.Require().Equal(aNEXBInitialBalance.Amount.Sub(math.NewInt(amount)).Sub(sendAndReceiveMsgFee), aNEXBAfterBalance.Amount)

			// check ibc aNEXB coins balance on Osmosis
			aNEXBIBCBalanceBefore := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), senderAcc, teststypes.aNEXBIbcdenom)
			s.Require().Equal(amount, aNEXBIBCBalanceBefore.Amount.Int64())

			// 2. Send aNEXB IBC coins from Osmosis to Nexa
			ibcCoinMeta := fmt.Sprintf("%s/%s", teststypes.aNEXBDenomtrace.Path, teststypes.aNEXBDenomtrace.BaseDenom)
			s.SendBackCoins(s.pathOsmosisNexa, s.IBCOsmosisChain, teststypes.aNEXBIbcdenom, amount, sender, receiver, 1, ibcCoinMeta)

			// check ibc aNEXB coins balance on Osmosis - should be zero
			aNEXBIBCSenderFinalBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), senderAcc, teststypes.aNEXBIbcdenom)
			s.Require().Equal(int64(0), aNEXBIBCSenderFinalBalance.Amount.Int64())

			// check aNEXB balance after transfer - should be equal to initial balance
			aNEXBFinalBalance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, utils.BaseDenom)

			totalFees := sendBackCoinsFee.Add(sendAndReceiveMsgFee)
			s.Require().Equal(aNEXBInitialBalance.Amount.Sub(totalFees), aNEXBFinalBalance.Amount)

			// check IBC Coin balance - should be zero
			ibcCoinsBalance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, teststypes.aNEXBIbcdenom)
			s.Require().Equal(int64(0), ibcCoinsBalance.Amount.Int64())

			// Check ERC20 balances - should be zero
			balanceTokenAfter := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
			s.Require().Equal(int64(0), balanceTokenAfter.Int64())
		})
		It("should transfer and convert original erc20", func() {
			uosmoInitialBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), senderAcc, "uosmo")

			// 1. Send 'uosmo' from Osmosis to Nexa
			s.SendAndReceiveMessage(s.pathOsmosisNexa, s.IBCOsmosisChain, "uosmo", amount, sender, receiver, 1, "")

			// validate 'uosmo' was transferred successfully and converted to ERC20
			balanceERC20Token := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
			s.Require().Equal(amount, balanceERC20Token.Int64())

			// 2. Transfer back the erc20 from Nexa to Osmosis
			ibcCoinMeta := fmt.Sprintf("%s/%s", teststypes.UosmoDenomtrace.Path, teststypes.UosmoDenomtrace.BaseDenom)
			s.SendBackCoins(s.pathOsmosisNexa, s.NexaChain, types.ModuleName+"/"+pair.GetERC20Contract().String(), amount, receiver, sender, 1, ibcCoinMeta)

			// after transfer, ERC-20 token balance should be zero
			balanceTokenAfter := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
			s.Require().Equal(int64(0), balanceTokenAfter.Int64())

			// check IBC Coin balance - should be zero
			ibcCoinsBalance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(int64(0), ibcCoinsBalance.Amount.Int64())

			// Final balance on Osmosis should be equal to initial balance
			uosmoFinalBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), senderAcc, "uosmo")
			s.Require().Equal(uosmoInitialBalance.Amount.Int64(), uosmoFinalBalance.Amount.Int64())
		})
	})
	Describe("Performing recovery with registered coin", func() {
		BeforeEach(func() {
			erc20params := types.DefaultParams()
			erc20params.EnableErc20 = true
			err := s.app.Erc20Keeper.SetParams(s.NexaChain.GetContext(), erc20params)
			s.Require().NoError(err)

			sender = s.IBCOsmosisChain.SenderAccount.GetAddress().String()
			// receiver address is on Osmosis Chain also,
			// but funds are transferred to this address in Nexa chain
			receiver = sender
			senderAcc = sdk.MustAccAddressFromBech32(sender)
			receiverAcc = sdk.MustAccAddressFromBech32(receiver)

			// Register uosmo pair
			pair, err = s.app.Erc20Keeper.RegisterCoin(s.NexaChain.GetContext(), osmoMeta)
			s.Require().NoError(err)
		})
		It("should recover and not convert uosmo to tokens", func() {
			uosmoInitialBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), senderAcc, "uosmo")

			// Send 'uosmo' to Osmosis address in Nexa Chain (locked funds)
			// sender_addr == receiver_addr
			s.SendAndReceiveMessage(s.pathOsmosisNexa, s.IBCOsmosisChain, "uosmo", amount, sender, receiver, 1, "")
			timeout := uint64(s.NexaChain.GetContext().BlockTime().Add(time.Hour * 4).Add(time.Second * -20).UnixNano())
			err := s.pathOsmosisNexa.RelayPacket(CreatePacket("10", "transfer/channel-0/uosmo", sender, receiver, "transfer", "channel-0", "transfer", "channel-0", 1, timeout))
			s.Require().NoError(err)

			s.IBCOsmosisChain.Coordinator.CommitNBlocks(s.IBCOsmosisChain, 10)
			// recovery should trigger and send back the funds to origin account
			// in the Osmosis Chain

			// ERC-20 balance should be zero
			balanceTokenAfter := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(receiverAcc.Bytes()))
			s.Require().Equal(int64(0), balanceTokenAfter.Int64())

			// IBC coin balance should be zero
			ibcCoinsBalance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), receiverAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(int64(0), ibcCoinsBalance.Amount.Int64())

			// validate that Osmosis address final balance == initial balance
			uosmoFinalBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), senderAcc, "uosmo")
			s.Require().Equal(uosmoInitialBalance.Amount.Int64(), uosmoFinalBalance.Amount.Int64())
		})
	})

	
	Describe("registered erc20", func() {
		BeforeEach(func() { //nolint:dupl
			erc20params := types.DefaultParams()
			erc20params.EnableErc20 = true
			err := s.app.Erc20Keeper.SetParams(s.NexaChain.GetContext(), erc20params)
			s.Require().NoError(err)

			receiver = s.IBCOsmosisChain.SenderAccount.GetAddress().String()
			sender = s.NexaChain.SenderAccount.GetAddress().String()
			receiverAcc = sdk.MustAccAddressFromBech32(receiver)
			senderAcc = sdk.MustAccAddressFromBech32(sender)

			// Register ERC20 pair
			addr, err := s.DeployContractToChain("testcoin", "tt", 18)
			s.Require().NoError(err)
			pair, err = s.app.Erc20Keeper.RegisterERC20(s.NexaChain.GetContext(), addr)
			s.Require().NoError(err)

			erc20Denomtrace = transfertypes.DenomTrace{
				Path:      "transfer/channel-0",
				BaseDenom: pair.Denom,
			}

			s.NexaChain.SenderAccount.SetSequence(s.NexaChain.SenderAccount.GetSequence() + 1) //nolint:errcheck
		})
		It("should convert erc20 ibc voucher to original erc20", func() {
			// Mint tokens and send to receiver
			_, err := s.app.Erc20Keeper.CallEVM(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, common.BytesToAddress(senderAcc.Bytes()), pair.GetERC20Contract(), true, "mint", common.BytesToAddress(senderAcc.Bytes()), big.NewInt(amount))
			s.Require().NoError(err)
			// Check Balance
			balanceToken := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())

			// Convert half of the available tokens
			msgConvertERC20 := types.NewMsgConvertERC20(
				sdk.NewInt(amount),
				senderAcc,
				pair.GetERC20Contract(),
				common.BytesToAddress(senderAcc.Bytes()),
			)

			err = msgConvertERC20.ValidateBasic()
			s.Require().NoError(err)
			// Use MsgConvertERC20 to convert the ERC20 to a Cosmos IBC Coin
			_, err = s.app.Erc20Keeper.ConvertERC20(sdk.WrapSDKContext(s.NexaChain.GetContext()), msgConvertERC20)
			s.Require().NoError(err)

			// Check Balance
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(int64(0), balanceToken.Int64())

			// IBC coin balance should be amount
			erc20CoinsBalance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, pair.Denom)
			s.Require().Equal(amount, erc20CoinsBalance.Amount.Int64())

			s.NexaChain.Coordinator.CommitBlock()

			// Attempt to send erc20 into ibc, should send without conversion
			s.SendBackCoins(s.pathOsmosisNexa, s.NexaChain, pair.Denom, amount, sender, receiver, 1, pair.Denom)
			s.IBCOsmosisChain.Coordinator.CommitBlock()

			// Check balance on the Osmosis chain
			erc20IBCBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), receiverAcc, erc20Denomtrace.IBCDenom())
			s.Require().Equal(amount, erc20IBCBalance.Amount.Int64())

			s.SendAndReceiveMessage(s.pathOsmosisNexa, s.IBCOsmosisChain, erc20Denomtrace.IBCDenom(), amount, receiver, sender, 1, erc20Denomtrace.GetFullDenomPath())
			// Check Balance
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())
		})
		It("should convert full available balance of erc20 coin to original erc20 token", func() {
			// Mint tokens and send to receiver
			_, err := s.app.Erc20Keeper.CallEVM(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, common.BytesToAddress(senderAcc.Bytes()), pair.GetERC20Contract(), true, "mint", common.BytesToAddress(senderAcc.Bytes()), big.NewInt(amount))
			s.Require().NoError(err)
			// Check Balance
			balanceToken := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())

			// Convert half of the available tokens
			msgConvertERC20 := types.NewMsgConvertERC20(
				sdk.NewInt(amount),
				senderAcc,
				pair.GetERC20Contract(),
				common.BytesToAddress(senderAcc.Bytes()),
			)

			err = msgConvertERC20.ValidateBasic()
			s.Require().NoError(err)
			// Use MsgConvertERC20 to convert the ERC20 to a Cosmos IBC Coin
			_, err = s.app.Erc20Keeper.ConvertERC20(sdk.WrapSDKContext(s.NexaChain.GetContext()), msgConvertERC20)
			s.Require().NoError(err)

			// Check Balance
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(int64(0), balanceToken.Int64())

			// erc20 coin balance should be amount
			erc20CoinsBalance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, pair.Denom)
			s.Require().Equal(amount, erc20CoinsBalance.Amount.Int64())

			s.NexaChain.Coordinator.CommitBlock()

			// Attempt to send erc20 into ibc, should send without conversion
			s.SendBackCoins(s.pathOsmosisNexa, s.NexaChain, pair.Denom, amount/2, sender, receiver, 1, pair.Denom)
			s.IBCOsmosisChain.Coordinator.CommitBlock()

			// Check balance on the Osmosis chain
			erc20IBCBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), receiverAcc, erc20Denomtrace.IBCDenom())
			s.Require().Equal(amount/2, erc20IBCBalance.Amount.Int64())

			s.SendAndReceiveMessage(s.pathOsmosisNexa, s.IBCOsmosisChain, erc20Denomtrace.IBCDenom(), amount/2, receiver, sender, 1, erc20Denomtrace.GetFullDenomPath())
			// Check Balance
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())

			// IBC coin balance should be zero
			erc20CoinsBalance = s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, pair.Denom)
			s.Require().Equal(int64(0), erc20CoinsBalance.Amount.Int64())
		})
		It("send native ERC-20 to osmosis, when sending back IBC coins should convert full balance back to erc20 token", func() {
			// Mint tokens and send to receiver
			_, err := s.app.Erc20Keeper.CallEVM(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, common.BytesToAddress(senderAcc.Bytes()), pair.GetERC20Contract(), true, "mint", common.BytesToAddress(senderAcc.Bytes()), big.NewInt(amount))
			s.Require().NoError(err)
			// Check Balance
			balanceToken := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())

			s.NexaChain.Coordinator.CommitBlock()

			// Attempt to send 1/2 of erc20 balance via ibc, should convert erc20 tokens to ibc coins and send the converted balance via IBC
			s.SendBackCoins(s.pathOsmosisNexa, s.NexaChain, types.ModuleName+"/"+pair.GetERC20Contract().String(), amount/2, sender, receiver, 1, "")
			s.IBCOsmosisChain.Coordinator.CommitBlock()

			// IBC coin balance should be zero
			erc20CoinsBalance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, pair.Denom)
			s.Require().Equal(int64(0), erc20CoinsBalance.Amount.Int64())

			// Check updated token Balance
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount/2, balanceToken.Int64())

			// Check balance on the Osmosis chain
			erc20IBCBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), receiverAcc, erc20Denomtrace.IBCDenom())
			s.Require().Equal(amount/2, erc20IBCBalance.Amount.Int64())

			// send back the IBC coins from Osmosis to Nexa
			s.SendAndReceiveMessage(s.pathOsmosisNexa, s.IBCOsmosisChain, erc20Denomtrace.IBCDenom(), amount/2, receiver, sender, 1, erc20Denomtrace.GetFullDenomPath())
			// Check Balance
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())

			// IBC coin balance should be zero
			erc20CoinsBalance = s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, pair.Denom)
			s.Require().Equal(int64(0), erc20CoinsBalance.Amount.Int64())
		})
	})
})

var _ = Describe("Convert outgoing ERC20 to IBC", Ordered, func() {
	var (
		sender, receiver string
		receiverAcc      sdk.AccAddress
		senderAcc        sdk.AccAddress
		amount           int64 = 10
		pair             *types.TokenPair
		erc20Denomtrace  transfertypes.DenomTrace
	)

	// Metadata to register OSMO with a Token Pair for testing
	osmoMeta := banktypes.Metadata{
		Description: "IBC Coin for IBC Osmosis Chain",
		Base:        teststypes.UosmoIbcdenom,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    teststypes.UosmoDenomtrace.BaseDenom,
				Exponent: 0,
			},
		},
		Name:    teststypes.UosmoIbcdenom,
		Symbol:  erc20Symbol,
		Display: teststypes.UosmoDenomtrace.BaseDenom,
	}

	BeforeEach(func() {
		s.suiteIBCTesting = true
		s.SetupTest()
		s.suiteIBCTesting = false
	})

	Describe("disabled params", func() {
		BeforeEach(func() {
			erc20params := types.DefaultParams()
			erc20params.EnableErc20 = true
			err := s.app.Erc20Keeper.SetParams(s.NexaChain.GetContext(), erc20params)
			s.Require().NoError(err)

			receiver = s.IBCOsmosisChain.SenderAccount.GetAddress().String()
			sender = s.NexaChain.SenderAccount.GetAddress().String()
			receiverAcc = sdk.MustAccAddressFromBech32(receiver)
			senderAcc = sdk.MustAccAddressFromBech32(sender)

			// Register ERC20 pair
			addr, err := s.DeployContractToChain("testcoin", "tt", 18)
			s.Require().NoError(err)
			pair, err = s.app.Erc20Keeper.RegisterERC20(s.NexaChain.GetContext(), addr)
			s.Require().NoError(err)
			s.NexaChain.Coordinator.CommitBlock()
			erc20params.EnableErc20 = false
			err = s.app.Erc20Keeper.SetParams(s.NexaChain.GetContext(), erc20params)
			s.Require().NoError(err)
		})
		It("should fail transfer and not convert to IBC", func() {
			// Mint tokens and send to receiver
			_, err := s.app.Erc20Keeper.CallEVM(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, common.BytesToAddress(senderAcc.Bytes()), pair.GetERC20Contract(), true, "mint", common.BytesToAddress(senderAcc.Bytes()), big.NewInt(amount))
			s.Require().NoError(err)
			// Check Balance
			balanceToken := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())

			path := s.pathOsmosisNexa
			originEndpoint := path.EndpointB
			destEndpoint := path.EndpointA
			originChain := s.NexaChain
			coin := pair.Denom
			transfer := transfertypes.NewFungibleTokenPacketData(pair.Denom, strconv.Itoa(int(amount*2)), sender, receiver, "")
			transferMsg := transfertypes.NewMsgTransfer(originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID, sdk.NewCoin(coin, sdk.NewInt(amount*2)), sender, receiver, timeoutHeight, 0, "")

			originChain.Coordinator.UpdateTimeForChain(originChain)
			denom := originChain.App.(*app.Nexa).StakingKeeper.BondDenom(originChain.GetContext())
			fee := sdk.Coins{sdk.NewInt64Coin(denom, ibctesting.DefaultFeeAmt)}

			_, _, err = ibctesting.SignAndDeliver(
				originChain.T,
				originChain.TxConfig,
				originChain.App.GetBaseApp(),
				[]sdk.Msg{transferMsg},
				fee,
				originChain.ChainID,
				[]uint64{originChain.SenderAccount.GetAccountNumber()},
				[]uint64{originChain.SenderAccount.GetSequence()},
				false, originChain.SenderPrivKey,
			)
			s.Require().Error(err)
			// NextBlock calls app.Commit()
			originChain.NextBlock()

			// increment sequence for successful transaction execution
			err = originChain.SenderAccount.SetSequence(originChain.SenderAccount.GetSequence() + 1)
			s.Require().NoError(err)
			originChain.Coordinator.IncrementTime()

			packet := channeltypes.NewPacket(transfer.GetBytes(), 1, originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID, destEndpoint.ChannelConfig.PortID, destEndpoint.ChannelID, timeoutHeight, 0)
			// Receive message on the counterparty side, and send ack
			err = path.RelayPacket(packet)
			s.Require().Error(err)

			// Check Balance didnt change
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())
		})
	})
	Describe("registered erc20", func() {
		BeforeEach(func() { //nolint:dupl
			erc20params := types.DefaultParams()
			erc20params.EnableErc20 = true
			err := s.app.Erc20Keeper.SetParams(s.NexaChain.GetContext(), erc20params)
			s.Require().NoError(err)

			receiver = s.IBCOsmosisChain.SenderAccount.GetAddress().String()
			sender = s.NexaChain.SenderAccount.GetAddress().String()
			receiverAcc = sdk.MustAccAddressFromBech32(receiver)
			senderAcc = sdk.MustAccAddressFromBech32(sender)

			// Register ERC20 pair
			addr, err := s.DeployContractToChain("testcoin", "tt", 18)
			s.Require().NoError(err)
			pair, err = s.app.Erc20Keeper.RegisterERC20(s.NexaChain.GetContext(), addr)
			s.Require().NoError(err)

			erc20Denomtrace = transfertypes.DenomTrace{
				Path:      "transfer/channel-0",
				BaseDenom: pair.Denom,
			}

			s.NexaChain.SenderAccount.SetSequence(s.NexaChain.SenderAccount.GetSequence() + 1) //nolint:errcheck
		})
		It("should transfer available balance", func() {
			// Mint tokens and send to receiver
			_, err := s.app.Erc20Keeper.CallEVM(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, common.BytesToAddress(senderAcc.Bytes()), pair.GetERC20Contract(), true, "mint", common.BytesToAddress(senderAcc.Bytes()), big.NewInt(amount*2))
			s.Require().NoError(err)
			// Check Balance
			balanceToken := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount*2, balanceToken.Int64())

			// Convert half of the available tokens
			msgConvertERC20 := types.NewMsgConvertERC20(
				sdk.NewInt(amount),
				senderAcc,
				pair.GetERC20Contract(),
				common.BytesToAddress(senderAcc.Bytes()),
			)

			err = msgConvertERC20.ValidateBasic()
			s.Require().NoError(err)
			// Use MsgConvertERC20 to convert the ERC20 to a Cosmos IBC Coin
			_, err = s.app.Erc20Keeper.ConvertERC20(sdk.WrapSDKContext(s.NexaChain.GetContext()), msgConvertERC20)
			s.Require().NoError(err)

			// Check Balance
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())

			// IBC coin balance should be amount
			erc20CoinsBalance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, pair.Denom)
			s.Require().Equal(amount, erc20CoinsBalance.Amount.Int64())

			s.NexaChain.Coordinator.CommitBlock()

			// Attempt to send erc20 into ibc, should send without conversion
			s.SendBackCoins(s.pathOsmosisNexa, s.NexaChain, pair.Denom, amount, sender, receiver, 1, pair.Denom)
			s.IBCOsmosisChain.Coordinator.CommitBlock()

			// Check balance on the Osmosis chain
			erc20IBCBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), receiverAcc, erc20Denomtrace.IBCDenom())
			s.Require().Equal(amount, erc20IBCBalance.Amount.Int64())
			// Check Balance
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())
		})
		It("should convert and transfer if no ibc balance", func() {
			// Mint tokens and send to receiver
			_, err := s.app.Erc20Keeper.CallEVM(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, common.BytesToAddress(senderAcc.Bytes()), pair.GetERC20Contract(), true, "mint", common.BytesToAddress(senderAcc.Bytes()), big.NewInt(amount))
			s.Require().NoError(err)

			// Check Balance
			balanceToken := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())

			// Attempt to send erc20 into ibc, should automatically convert
			s.SendBackCoins(s.pathOsmosisNexa, s.NexaChain, pair.Denom, amount, sender, receiver, 1, pair.Denom)

			s.NexaChain.Coordinator.CommitBlock()
			// Check balance of erc20 depleted
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(int64(0), balanceToken.Int64())

			// Check balance received on the Osmosis chain
			ibcOsmosBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), receiverAcc, erc20Denomtrace.IBCDenom())
			s.Require().Equal(amount, ibcOsmosBalance.Amount.Int64())
		})
		It("should fail if balance is not enough", func() {
			// Mint tokens and send to receiver
			_, err := s.app.Erc20Keeper.CallEVM(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, common.BytesToAddress(senderAcc.Bytes()), pair.GetERC20Contract(), true, "mint", common.BytesToAddress(senderAcc.Bytes()), big.NewInt(amount))
			s.Require().NoError(err)

			// Check Balance
			balanceToken := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())

			// Attempt to send that will fail because balance is not enough
			path := s.pathOsmosisNexa
			originEndpoint := path.EndpointB
			originChain := s.NexaChain
			coin := pair.Denom
			transferMsg := transfertypes.NewMsgTransfer(originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID, sdk.NewCoin(coin, sdk.NewInt(amount*2)), sender, receiver, timeoutHeight, 0, "")

			originChain.Coordinator.UpdateTimeForChain(originChain)

			denom := originChain.App.(*app.Nexa).StakingKeeper.BondDenom(originChain.GetContext())
			fee := sdk.Coins{sdk.NewInt64Coin(denom, ibctesting.DefaultFeeAmt)}

			_, _, err = ibctesting.SignAndDeliver(
				originChain.T,
				originChain.TxConfig,
				originChain.App.GetBaseApp(),
				[]sdk.Msg{transferMsg},
				fee,
				originChain.ChainID,
				[]uint64{originChain.SenderAccount.GetAccountNumber()},
				[]uint64{originChain.SenderAccount.GetSequence()},
				false, originChain.SenderPrivKey,
			)

			// Require a failing transfer
			s.Require().Error(err)
			// NextBlock calls app.Commit()
			originChain.NextBlock()
			originChain.Coordinator.IncrementTime()

			// Check Balance didnt change
			ibcOsmosBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), receiverAcc, erc20Denomtrace.IBCDenom())
			s.Require().Equal(int64(0), ibcOsmosBalance.Amount.Int64())
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())
		})
		It("should timeout and reconvert to ERC20", func() {
			// Mint tokens and send to receiver
			_, err := s.app.Erc20Keeper.CallEVM(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, common.BytesToAddress(senderAcc.Bytes()), pair.GetERC20Contract(), true, "mint", common.BytesToAddress(senderAcc.Bytes()), big.NewInt(amount))
			s.Require().NoError(err)

			// Check Balance
			balanceToken := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())

			// Attempt to send erc20 into ibc, should automatically convert
			// Send message that will timeout
			path := s.pathOsmosisNexa
			originEndpoint := path.EndpointB
			destEndpoint := path.EndpointA
			originChain := s.NexaChain
			coin := pair.Denom
			currentTime := s.NexaChain.Coordinator.CurrentTime
			timeout := uint64(currentTime.Unix() * 1000000000)
			transferMsg := transfertypes.NewMsgTransfer(originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID,
				sdk.NewCoin(coin, sdk.NewInt(amount)), sender, receiver, timeoutHeight, timeout, "")

			originChain.Coordinator.UpdateTimeForChain(originChain)

			denom := originChain.App.(*app.Nexa).StakingKeeper.BondDenom(originChain.GetContext())
			fee := sdk.Coins{sdk.NewInt64Coin(denom, ibctesting.DefaultFeeAmt)}

			_, _, err = ibctesting.SignAndDeliver(
				originChain.T,
				originChain.TxConfig,
				originChain.App.GetBaseApp(),
				[]sdk.Msg{transferMsg},
				fee,
				originChain.ChainID,
				[]uint64{originChain.SenderAccount.GetAccountNumber()},
				[]uint64{originChain.SenderAccount.GetSequence()},
				true, originChain.SenderPrivKey,
			)
			s.Require().NoError(err)

			// Check balance of erc20 depleted (converted to IBC coin)
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(int64(0), balanceToken.Int64())

			// NextBlock calls app.Commit()
			originChain.NextBlock()

			// increment sequence for successful transaction execution
			err = originChain.SenderAccount.SetSequence(originChain.SenderAccount.GetSequence() + 1)
			s.Require().NoError(err)

			// Increment time so packet will timeout
			originChain.Coordinator.IncrementTime()
			s.IBCOsmosisChain.Coordinator.CommitBlock(s.IBCOsmosisChain)

			// Recreate the packet that was sent
			transfer := transfertypes.NewFungibleTokenPacketData(pair.Denom, strconv.Itoa(int(amount)), sender, receiver, "")
			packet := channeltypes.NewPacket(transfer.GetBytes(), 1, originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID, destEndpoint.ChannelConfig.PortID, destEndpoint.ChannelID, timeoutHeight, timeout)

			// need to update nexa chain to prove missing ack
			err = path.EndpointB.UpdateClient()
			s.Require().NoError(err)
			// Receive timeout
			err = path.EndpointB.TimeoutPacket(packet)
			s.Require().NoError(err)
			originChain.NextBlock()

			// Check that balance was reconverted to ERC20 and refunded to sender
			balanceToken = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceToken.Int64())
		})
	})
	Describe("registered coin", func() {
		BeforeEach(func() {
			receiver = s.IBCOsmosisChain.SenderAccount.GetAddress().String()
			sender = s.NexaChain.SenderAccount.GetAddress().String()
			receiverAcc = sdk.MustAccAddressFromBech32(receiver)
			senderAcc = sdk.MustAccAddressFromBech32(sender)

			erc20params := types.DefaultParams()
			erc20params.EnableErc20 = false
			err := s.app.Erc20Keeper.SetParams(s.NexaChain.GetContext(), erc20params)
			s.Require().NoError(err)

			// Send from osmosis to Nexa
			s.SendAndReceiveMessage(s.pathOsmosisNexa, s.IBCOsmosisChain, "uosmo", amount, receiver, sender, 1, "")
			s.NexaChain.Coordinator.CommitBlock(s.NexaChain)
			erc20params.EnableErc20 = true
			err = s.app.Erc20Keeper.SetParams(s.NexaChain.GetContext(), erc20params)
			s.Require().NoError(err)

			// Register uosmo pair
			pair, err = s.app.Erc20Keeper.RegisterCoin(s.NexaChain.GetContext(), osmoMeta)
			s.Require().NoError(err)
		})
		It("should convert erc20 to ibc vouched and transfer", func() {
			uosmoInitialBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), receiverAcc, "uosmo")

			balance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(amount, balance.Amount.Int64())

			// Convert ibc vouchers to erc20 tokens
			msgConvertCoin := types.NewMsgConvertCoin(
				sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
				common.BytesToAddress(senderAcc.Bytes()),
				senderAcc,
			)

			err := msgConvertCoin.ValidateBasic()
			s.Require().NoError(err)
			// Use MsgConvertERC20 to convert the ERC20 to a Cosmos IBC Coin
			_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.NexaChain.GetContext()), msgConvertCoin)
			s.Require().NoError(err)

			s.NexaChain.Coordinator.CommitBlock()

			// Attempt to send erc20 tokens to osmosis and convert automatically
			s.SendBackCoins(s.pathOsmosisNexa, s.NexaChain, pair.Denom, amount, sender, receiver, 1, teststypes.UosmoDenomtrace.GetFullDenomPath())
			s.IBCOsmosisChain.Coordinator.CommitBlock()
			// Check balance on the Osmosis chain
			uosmoBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), receiverAcc, "uosmo")
			s.Require().Equal(uosmoInitialBalance.Amount.Int64()+amount, uosmoBalance.Amount.Int64())
		})
		It("should transfer available balance", func() {
			uosmoInitialBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), receiverAcc, "uosmo")

			balance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(amount, balance.Amount.Int64())

			// Attempt to send erc20 tokens to osmosis and convert automatically
			s.SendBackCoins(s.pathOsmosisNexa, s.NexaChain, pair.Denom, amount, sender, receiver, 1, teststypes.UosmoDenomtrace.GetFullDenomPath())
			s.IBCOsmosisChain.Coordinator.CommitBlock()
			// Check balance on the Osmosis chain
			uosmoBalance := s.IBCOsmosisChain.GetSimApp().BankKeeper.GetBalance(s.IBCOsmosisChain.GetContext(), receiverAcc, "uosmo")
			s.Require().Equal(uosmoInitialBalance.Amount.Int64()+amount, uosmoBalance.Amount.Int64())
		})

		It("should timeout and reconvert coins", func() {
			balance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(amount, balance.Amount.Int64())

			// Convert ibc vouchers to erc20 tokens
			msgConvertCoin := types.NewMsgConvertCoin(
				sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
				common.BytesToAddress(senderAcc.Bytes()),
				senderAcc,
			)
			err := msgConvertCoin.ValidateBasic()
			s.Require().NoError(err)

			// Use MsgConvertERC20 to convert the ERC20 to a Cosmos IBC Coin
			_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.NexaChain.GetContext()), msgConvertCoin)
			s.Require().NoError(err)

			s.NexaChain.Coordinator.CommitBlock()

			// Send message that will timeout
			path := s.pathOsmosisNexa
			originEndpoint := path.EndpointB
			destEndpoint := path.EndpointA
			originChain := s.NexaChain
			coin := pair.Denom
			currentTime := s.NexaChain.Coordinator.CurrentTime
			timeout := uint64(currentTime.Unix() * 1000000000)
			transferMsg := transfertypes.NewMsgTransfer(originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID,
				sdk.NewCoin(coin, sdk.NewInt(amount)), sender, receiver, timeoutHeight, timeout, "")

			originChain.Coordinator.UpdateTimeForChain(originChain)

			denom := originChain.App.(*app.Nexa).StakingKeeper.BondDenom(originChain.GetContext())
			fee := sdk.Coins{sdk.NewInt64Coin(denom, ibctesting.DefaultFeeAmt)}

			_, _, err = ibctesting.SignAndDeliver(
				originChain.T,
				originChain.TxConfig,
				originChain.App.GetBaseApp(),
				[]sdk.Msg{transferMsg},
				fee,
				originChain.ChainID,
				[]uint64{originChain.SenderAccount.GetAccountNumber()},
				[]uint64{originChain.SenderAccount.GetSequence()},
				true, originChain.SenderPrivKey,
			)
			s.Require().NoError(err)

			// check ERC20 balance was converted to ibc and sent
			balanceERC20TokenAfter := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(int64(0), balanceERC20TokenAfter.Int64())

			// NextBlock calls app.Commit()
			originChain.NextBlock()

			// increment sequence for successful transaction execution
			err = originChain.SenderAccount.SetSequence(originChain.SenderAccount.GetSequence() + 1)
			s.Require().NoError(err)

			// Increment time so packet will timeout
			originChain.Coordinator.IncrementTime()
			s.IBCOsmosisChain.Coordinator.CommitBlock(s.IBCOsmosisChain)

			// Recreate the packet that was sent
			transfer := transfertypes.NewFungibleTokenPacketData(teststypes.UosmoDenomtrace.GetFullDenomPath(), strconv.Itoa(int(amount)), sender, receiver, "")
			packet := channeltypes.NewPacket(transfer.GetBytes(), 1, originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID, destEndpoint.ChannelConfig.PortID, destEndpoint.ChannelID, timeoutHeight, timeout)

			// need to update nexa chain to prove missing ack
			err = path.EndpointB.UpdateClient()
			s.Require().NoError(err)
			// Receive timeout
			err = path.EndpointB.TimeoutPacket(packet)
			s.Require().NoError(err)
			originChain.NextBlock()

			// Check that balance was reconverted
			balance = s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(int64(0), balance.Amount.Int64())

			balanceERC20TokenAfter = s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceERC20TokenAfter.Int64())
		})
		It("should error and reconvert coins", func() {
			receiverAcc = s.IBCCosmosChain.GetSimApp().AccountKeeper.GetModuleAddress("distribution")
			receiver = receiverAcc.String()
			s.IBCOsmosisChain.GetSimApp().BankKeeper.BlockedAddr(receiverAcc)

			balance := s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(amount, balance.Amount.Int64())

			// Convert ibc vouchers to erc20 tokens
			msgConvertCoin := types.NewMsgConvertCoin(
				sdk.NewCoin(pair.Denom, sdk.NewInt(amount)),
				common.BytesToAddress(senderAcc.Bytes()),
				senderAcc,
			)
			err := msgConvertCoin.ValidateBasic()
			s.Require().NoError(err)

			// Use MsgConvertERC20 to convert the ERC20 to a Cosmos IBC Coin
			_, err = s.app.Erc20Keeper.ConvertCoin(sdk.WrapSDKContext(s.NexaChain.GetContext()), msgConvertCoin)
			s.Require().NoError(err)

			s.NexaChain.Coordinator.CommitBlock()

			// Send message that will timeout
			path := s.pathOsmosisNexa
			originEndpoint := path.EndpointB
			destEndpoint := path.EndpointA
			originChain := s.NexaChain
			coin := pair.Denom
			timeout := uint64(0)
			transferMsg := transfertypes.NewMsgTransfer(originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID,
				sdk.NewCoin(coin, sdk.NewInt(amount)), sender, receiver, timeoutHeight, timeout, "")

			_, err = ibctesting.SendMsgs(originChain, ibctesting.DefaultFeeAmt, transferMsg)
			s.Require().NoError(err) // message committed

			// Recreate the packet that was sent
			transfer := transfertypes.NewFungibleTokenPacketData(teststypes.UosmoDenomtrace.GetFullDenomPath(), strconv.Itoa(int(amount)), sender, receiver, "")
			packet := channeltypes.NewPacket(transfer.GetBytes(), 1, originEndpoint.ChannelConfig.PortID, originEndpoint.ChannelID, destEndpoint.ChannelConfig.PortID, destEndpoint.ChannelID, timeoutHeight, 0)

			// Receive message on the counterparty side, and send ack
			err = path.RelayPacket(packet)
			s.Require().NoError(err)

			balance = s.app.BankKeeper.GetBalance(s.NexaChain.GetContext(), senderAcc, teststypes.UosmoIbcdenom)
			s.Require().Equal(int64(0), balance.Amount.Int64())

			balanceERC20TokenAfter := s.app.Erc20Keeper.BalanceOf(s.NexaChain.GetContext(), contracts.ERC20MinterBurnerDecimalsContract.ABI, pair.GetERC20Contract(), common.BytesToAddress(senderAcc.Bytes()))
			s.Require().Equal(amount, balanceERC20TokenAfter.Int64())
		})
	})
})
