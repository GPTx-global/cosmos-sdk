package testutil

import (
	"fmt"

	"github.com/stretchr/testify/suite"
	tmcli "github.com/tendermint/tendermint/libs/cli"

	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/client/cli"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

type RatioDistributionTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

func NewRatioDistributionTestSuite(cfg network.Config) *RatioDistributionTestSuite {
	return &RatioDistributionTestSuite{cfg: cfg}
}

// SetupSuite creates a new network for _each_ integration test. We create a new
// network for each test because there are some state modifications that are
// needed to be made in order to make useful queries. However, we don't want
// these state changes to be present in other tests.
func (s *RatioDistributionTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")

	cfg := network.DefaultConfig()
	cfg.NumValidators = 1
	s.cfg = cfg

	genesisState := s.cfg.GenesisState
	var mintData minttypes.GenesisState
	s.Require().NoError(s.cfg.Codec.UnmarshalJSON(genesisState[minttypes.ModuleName], &mintData))

	// set inflation to 0%
	inflation := sdk.MustNewDecFromStr("0.0")
	mintData.Minter.Inflation = inflation
	mintData.Params.InflationMin = inflation
	mintData.Params.InflationMax = inflation

	mintDataBz, err := s.cfg.Codec.MarshalJSON(&mintData)
	s.Require().NoError(err)
	genesisState[minttypes.ModuleName] = mintDataBz

	// set distribution genesis
	var distData distrtypes.GenesisState
	s.Require().NoError(s.cfg.Codec.UnmarshalJSON(genesisState[distrtypes.ModuleName], &distData))
	distData.ModeratorAddress = distModeratorAddr
	distData.BaseAddress = distBaseAddr

	distDataBz, err := s.cfg.Codec.MarshalJSON(&distData)
	s.Require().NoError(err)
	genesisState[distrtypes.ModuleName] = distDataBz

	// set balance for test addresses
	var bankData banktypes.GenesisState
	s.Require().NoError(s.cfg.Codec.UnmarshalJSON(genesisState[banktypes.ModuleName], &bankData))
	bankData.Balances = append(bankData.Balances,
		banktypes.Balance{Address: distModeratorAddr, Coins: sdk.NewCoins(sdk.NewCoin(cfg.BondDenom, sdk.NewInt(1000000)))},
	)

	bankDataBz, err := s.cfg.Codec.MarshalJSON(&bankData)
	s.Require().NoError(err)
	genesisState[banktypes.ModuleName] = bankDataBz

	// update the genesis state
	s.cfg.GenesisState = genesisState

	s.network, err = network.New(s.T(), s.T().TempDir(), s.cfg)
	s.Require().NoError(err)

	_, err = s.network.WaitForHeight(1)
	s.Require().NoError(err)

}

// TearDownSuite cleans up the curret test network after _each_ test.
func (s *RatioDistributionTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite1")
	s.network.Cleanup()
}

func (s *RatioDistributionTestSuite) TestFeeDistribution() {
	// reset the suite
	// s.TearDownSuite()
	// s.SetupSuite()

	val := s.network.Validators[0]
	clientCtx := val.ClientCtx

	s.Run("correct distribution", func() {
		var ratio distrtypes.QueryRatioResponse
		var oldTS banktypes.QueryTotalSupplyResponse
		var newTS banktypes.QueryTotalSupplyResponse
		var oldBaseBalance banktypes.QueryAllBalancesResponse
		var newBaseBalance banktypes.QueryAllBalancesResponse
		var oldCommPool distrtypes.QueryCommunityPoolResponse
		var newCommPool distrtypes.QueryCommunityPoolResponse
		var oldRewards distrtypes.QueryDelegationRewardsResponse
		var newRewards distrtypes.QueryDelegationRewardsResponse
		var oldValCommission distrtypes.ValidatorAccumulatedCommission
		var newValCommission distrtypes.ValidatorAccumulatedCommission

		argJson := fmt.Sprintf("--%s=json", tmcli.OutputFlag)
		argH5 := fmt.Sprintf("--%s=5", flags.FlagHeight)
		argH10 := fmt.Sprintf("--%s=10", flags.FlagHeight)

		// get the ratio
		cmd := cli.GetCmdQueryRatio()
		out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{argJson})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &ratio), out.String())

		// get the old total supply
		cmd = bankcli.GetCmdQueryTotalSupply()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{argJson, argH5})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &oldTS), out.String())

		// get the old base address balance
		cmd = bankcli.GetBalancesCmd()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{distBaseAddr, argJson, argH5})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &oldBaseBalance), out.String())

		// get the old community pool
		cmd = cli.GetCmdQueryCommunityPool()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{argJson, argH5})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &oldCommPool), out.String())

		// get the old rewards
		cmd = cli.GetCmdQueryDelegatorRewards()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{val.Address.String(), sdk.ValAddress(val.Address).String(), argJson, argH5})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &oldRewards), out.String())

		// get the old validator commission
		cmd = cli.GetCmdQueryValidatorCommission()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{sdk.ValAddress(val.Address).String(), argJson, argH5})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &oldValCommission), out.String())

		// bank send
		argsTx := []string{
			val.Address.String(), changeAddr, "1000" + s.network.Config.BondDenom,
			fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
			fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
			fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
			fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(298))).String()),
		}
		cmd = bankcli.NewSendTxCmd()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, argsTx)
		s.Require().NoError(err)
		s.Require().Contains(out.String(), `"code":0`)

		// wait for two more blocks so the fee is distributed
		// s.network.WaitForNextBlock()
		s.network.WaitForNextBlock()
		s.network.WaitForNextBlock()

		// get the new total supply
		cmd = bankcli.GetCmdQueryTotalSupply()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{argJson, argH10})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &newTS), out.String())

		// get the new base address balance
		cmd = bankcli.GetBalancesCmd()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{distBaseAddr, argJson, argH10})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &newBaseBalance), out.String())

		// get the new community pool
		cmd = cli.GetCmdQueryCommunityPool()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{argJson, argH10})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &newCommPool), out.String())

		// get the new rewards
		cmd = cli.GetCmdQueryDelegatorRewards()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{val.Address.String(), sdk.ValAddress(val.Address).String(), argJson, argH10})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &newRewards), out.String())

		// get the new validator commission
		cmd = cli.GetCmdQueryValidatorCommission()
		out, err = clitestutil.ExecTestCLICmd(clientCtx, cmd, []string{sdk.ValAddress(val.Address).String(), argJson, argH10})
		s.Require().NoError(err)
		s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &newValCommission), out.String())

		// test 1/3 burn
		// fee: 298stake => burn: 99stake
		s.Require().Equal(oldTS.Supply.String(),
			newTS.Supply.
				Add(sdk.NewCoin(s.network.Config.BondDenom, sdk.NewInt(99))).String())

		// test 1/3 base address fee
		// fee: 298stake => base: 99stake
		s.Require().Equal(oldBaseBalance.Balances.String(),
			newBaseBalance.Balances.
				Sub(sdk.NewCoin(s.network.Config.BondDenom, sdk.NewInt(99))).String())

		// test 1/3 staking rewards
		// fee: 298stake => staking_rewards: 100stake

		// community pool:
		// staking_rewards: 100stake => pool: 2stake
		s.Require().Equal(oldCommPool.Pool.String(),
			newCommPool.Pool.
				Sub(sdk.NewDecCoins(sdk.NewDecCoin(s.network.Config.BondDenom, sdk.NewInt(2)))).String())

		// delegator rewards
		// staking_rewards: 100stake => delegators: 49stake
		s.Require().Equal(oldRewards.Rewards.String(),
			newRewards.Rewards.
				Sub(sdk.NewDecCoins(sdk.NewDecCoin(s.network.Config.BondDenom, sdk.NewInt(49)))).String())

		// validator commission
		// staking_rewards: 100stake => validators: 49stake
		s.Require().Equal(oldValCommission.Commission.String(),
			newValCommission.Commission.
				Sub(sdk.NewDecCoins(sdk.NewDecCoin(s.network.Config.BondDenom, sdk.NewInt(49)))).String())

	})
}
