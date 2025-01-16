package testutil

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/suite"
	tmcli "github.com/tendermint/tendermint/libs/cli"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankcli "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/client/cli"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

type IntegrationTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

var (
	distModeratorAddr = "cosmos10rc3qc4vynurg7f6etfhuhnmq6v7v03w7m2p39"
	distBaseAddr      = "cosmos10rc3qc4vynurg7f6etfhuhnmq6v7v03w7m2p39" //"cosmos139f7kncmglres2nf3h4hc4tade85ekfr8sulz5"
	changeAddr        = "cosmos1gtt8clsfjlyupuc92sl7432lc2a94na87d6guc"
	distModeratorMnic = "charge gloom capital outdoor ride mixture barely virus better depth admit speed turtle broccoli air find rib adult bid stock bar wreck amazing resist"
	// changeAddrMnic    = "unfold rotate test false round multiply measure catch pumpkin leaf mystery boil honey bridge toss gold enforce sort will marriage walk evidence task stairs"
)

func NewIntegrationTestSuite(cfg network.Config) *IntegrationTestSuite {
	return &IntegrationTestSuite{cfg: cfg}
}

// SetupSuite creates a new network for _each_ integration test. We create a new
// network for each test because there are some state modifications that are
// needed to be made in order to make useful queries. However, we don't want
// these state changes to be present in other tests.
func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")

	cfg := network.DefaultConfig()
	cfg.NumValidators = 1
	s.cfg = cfg

	genesisState := s.cfg.GenesisState
	var mintData minttypes.GenesisState
	s.Require().NoError(s.cfg.Codec.UnmarshalJSON(genesisState[minttypes.ModuleName], &mintData))

	inflation := sdk.MustNewDecFromStr("1.0")
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

	s.addSignerKey("moderator", distModeratorAddr, distModeratorMnic)

}

// TearDownSuite cleans up the curret test network after _each_ test.
func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite1")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestGetCmdQueryParams() {
	val := s.network.Validators[0]

	testCases := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			"json output",
			[]string{fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
			`{"community_tax":"0.020000000000000000","base_proposer_reward":"0.010000000000000000","bonus_proposer_reward":"0.040000000000000000","withdraw_addr_enabled":true}`,
		},
		{
			"text output",
			[]string{fmt.Sprintf("--%s=text", tmcli.OutputFlag)},
			`base_proposer_reward: "0.010000000000000000"
bonus_proposer_reward: "0.040000000000000000"
community_tax: "0.020000000000000000"
withdraw_addr_enabled: true`,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryParams()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			s.Require().NoError(err)
			s.Require().Equal(tc.expectedOutput, strings.TrimSpace(out.String()))
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdQueryValidatorDistributionInfo() {
	val := s.network.Validators[0]

	testCases := []struct {
		name   string
		args   []string
		expErr bool
	}{
		{
			"invalid val address",
			[]string{"invalid address", fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
			true,
		},
		{
			"json output",
			[]string{val.ValAddress.String(), fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
			false,
		},
		{
			"text output",
			[]string{val.ValAddress.String(), fmt.Sprintf("--%s=text", tmcli.OutputFlag)},
			false,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryValidatorDistributionInfo()
			clientCtx := val.ClientCtx

			_, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdQueryValidatorOutstandingRewards() {
	val := s.network.Validators[0]

	_, err := s.network.WaitForHeight(4)
	s.Require().NoError(err)

	testCases := []struct {
		name           string
		args           []string
		expectErr      bool
		expectedOutput string
	}{
		{
			"invalid validator address",
			[]string{
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				"foo",
			},
			true,
			"",
		},
		{
			"json output",
			[]string{
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				sdk.ValAddress(val.Address).String(),
				fmt.Sprintf("--%s=json", tmcli.OutputFlag),
			},
			false,
			`{"rewards":[{"denom":"stake","amount":"79.380000000000000000"}]}`,
		},
		{
			"text output",
			[]string{
				fmt.Sprintf("--%s=text", tmcli.OutputFlag),
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				sdk.ValAddress(val.Address).String(),
			},
			false,
			`rewards:
- amount: "79.380000000000000000"
  denom: stake`,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryValidatorOutstandingRewards()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedOutput, strings.TrimSpace(out.String()))
			}
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdQueryValidatorCommission() {
	val := s.network.Validators[0]

	_, err := s.network.WaitForHeight(4)
	s.Require().NoError(err)

	testCases := []struct {
		name           string
		args           []string
		expectErr      bool
		expectedOutput string
	}{
		{
			"invalid validator address",
			[]string{
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				"foo",
			},
			true,
			"",
		},
		{
			"json output",
			[]string{
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				sdk.ValAddress(val.Address).String(),
				fmt.Sprintf("--%s=json", tmcli.OutputFlag),
			},
			false,
			`{"commission":[{"denom":"stake","amount":"39.690000000000000000"}]}`,
		},
		{
			"text output",
			[]string{
				fmt.Sprintf("--%s=text", tmcli.OutputFlag),
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				sdk.ValAddress(val.Address).String(),
			},
			false,
			`commission:
- amount: "39.690000000000000000"
  denom: stake`,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryValidatorCommission()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedOutput, strings.TrimSpace(out.String()))
			}
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdQueryValidatorSlashes() {
	val := s.network.Validators[0]

	_, err := s.network.WaitForHeight(4)
	s.Require().NoError(err)

	testCases := []struct {
		name           string
		args           []string
		expectErr      bool
		expectedOutput string
	}{
		{
			"invalid validator address",
			[]string{
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				"foo", "1", "3",
			},
			true,
			"",
		},
		{
			"invalid start height",
			[]string{
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				sdk.ValAddress(val.Address).String(), "-1", "3",
			},
			true,
			"",
		},
		{
			"invalid end height",
			[]string{
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				sdk.ValAddress(val.Address).String(), "1", "-3",
			},
			true,
			"",
		},
		{
			"json output",
			[]string{
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				sdk.ValAddress(val.Address).String(), "1", "3",
				fmt.Sprintf("--%s=json", tmcli.OutputFlag),
			},
			false,
			"{\"slashes\":[],\"pagination\":{\"next_key\":null,\"total\":\"0\"}}",
		},
		{
			"text output",
			[]string{
				fmt.Sprintf("--%s=text", tmcli.OutputFlag),
				fmt.Sprintf("--%s=3", flags.FlagHeight),
				sdk.ValAddress(val.Address).String(), "1", "3",
			},
			false,
			"pagination:\n  next_key: null\n  total: \"0\"\nslashes: []",
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryValidatorSlashes()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedOutput, strings.TrimSpace(out.String()))
			}
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdQueryDelegatorRewards() {
	val := s.network.Validators[0]
	addr := val.Address
	valAddr := sdk.ValAddress(addr)

	_, err := s.network.WaitForHeightWithTimeout(11, time.Minute)
	s.Require().NoError(err)

	testCases := []struct {
		name           string
		args           []string
		expectErr      bool
		expectedOutput string
	}{
		{
			"invalid delegator address",
			[]string{
				fmt.Sprintf("--%s=5", flags.FlagHeight),
				"foo", valAddr.String(),
			},
			true,
			"",
		},
		{
			"invalid validator address",
			[]string{
				fmt.Sprintf("--%s=5", flags.FlagHeight),
				addr.String(), "foo",
			},
			true,
			"",
		},
		{
			"json output",
			[]string{
				fmt.Sprintf("--%s=5", flags.FlagHeight),
				addr.String(),
				fmt.Sprintf("--%s=json", tmcli.OutputFlag),
			},
			false,
			fmt.Sprintf(`{"rewards":[{"validator_address":"%s","reward":[{"denom":"stake","amount":"66.150000000000000000"}]}],"total":[{"denom":"stake","amount":"66.150000000000000000"}]}`, valAddr.String()),
		},
		{
			"json output (specific validator)",
			[]string{
				fmt.Sprintf("--%s=5", flags.FlagHeight),
				addr.String(), valAddr.String(),
				fmt.Sprintf("--%s=json", tmcli.OutputFlag),
			},
			false,
			`{"rewards":[{"denom":"stake","amount":"66.150000000000000000"}]}`,
		},
		{
			"text output",
			[]string{
				fmt.Sprintf("--%s=text", tmcli.OutputFlag),
				fmt.Sprintf("--%s=5", flags.FlagHeight),
				addr.String(),
			},
			false,
			fmt.Sprintf(`rewards:
- reward:
  - amount: "66.150000000000000000"
    denom: stake
  validator_address: %s
total:
- amount: "66.150000000000000000"
  denom: stake`, valAddr.String()),
		},
		{
			"text output (specific validator)",
			[]string{
				fmt.Sprintf("--%s=text", tmcli.OutputFlag),
				fmt.Sprintf("--%s=5", flags.FlagHeight),
				addr.String(), valAddr.String(),
			},
			false,
			`rewards:
- amount: "66.150000000000000000"
  denom: stake`,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryDelegatorRewards()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().Equal(tc.expectedOutput, strings.TrimSpace(out.String()))
			}
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdQueryCommunityPool() {
	val := s.network.Validators[0]

	_, err := s.network.WaitForHeight(4)
	s.Require().NoError(err)

	testCases := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			"json output",
			[]string{fmt.Sprintf("--%s=3", flags.FlagHeight), fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
			`{"pool":[{"denom":"stake","amount":"1.620000000000000000"}]}`,
		},
		{
			"text output",
			[]string{fmt.Sprintf("--%s=text", tmcli.OutputFlag), fmt.Sprintf("--%s=3", flags.FlagHeight)},
			`pool:
- amount: "1.620000000000000000"
  denom: stake`,
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryCommunityPool()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			s.Require().NoError(err)
			s.Require().Equal(tc.expectedOutput, strings.TrimSpace(out.String()))
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdQueryRatio() {

	val := s.network.Validators[0]

	testCases := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			"json output",
			[]string{fmt.Sprintf("--%s=json", tmcli.OutputFlag), fmt.Sprintf("--%s=3", flags.FlagHeight)},
			`{"ratio":{"staking_rewards":"0.333333333333333334","base":"0.333333333333333333","burn":"0.333333333333333333"}}`,
		},
		{
			"text output",
			[]string{fmt.Sprintf("--%s=text", tmcli.OutputFlag), fmt.Sprintf("--%s=3", flags.FlagHeight)},
			"ratio:\n  base: \"0.333333333333333333\"\n  burn: \"0.333333333333333333\"\n  staking_rewards: \"0.333333333333333334\"",
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryRatio()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			s.Require().NoError(err)
			s.Require().Equal(tc.expectedOutput, strings.TrimSpace(out.String()))
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdQueryModerator() {

	val := s.network.Validators[0]

	testCases := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			"json output",
			[]string{fmt.Sprintf("--%s=json", tmcli.OutputFlag), fmt.Sprintf("--%s=3", flags.FlagHeight)},
			fmt.Sprintf(`{"moderator_address":"%s"}`, distModeratorAddr),
		},
		{
			"text output",
			[]string{fmt.Sprintf("--%s=text", tmcli.OutputFlag), fmt.Sprintf("--%s=3", flags.FlagHeight)},
			fmt.Sprintf(`moderator_address: %s`, distModeratorAddr),
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryModerator()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			s.Require().NoError(err)
			s.Require().Equal(tc.expectedOutput, strings.TrimSpace(out.String()))
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdQueryBaseAddress() {

	val := s.network.Validators[0]

	testCases := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			"json output",
			[]string{fmt.Sprintf("--%s=json", tmcli.OutputFlag), fmt.Sprintf("--%s=3", flags.FlagHeight)},
			fmt.Sprintf(`{"base_address":"%s"}`, distBaseAddr),
		},
		{
			"text output",
			[]string{fmt.Sprintf("--%s=text", tmcli.OutputFlag), fmt.Sprintf("--%s=3", flags.FlagHeight)},
			fmt.Sprintf(`base_address: %s`, distBaseAddr),
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdQueryBaseAddress()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			s.Require().NoError(err)
			s.Require().Equal(tc.expectedOutput, strings.TrimSpace(out.String()))
		})
	}
}

func (s *IntegrationTestSuite) TestNewWithdrawRewardsCmd() {
	// reset the suite
	s.TearDownSuite()
	s.SetupSuite()

	val := s.network.Validators[0]

	testCases := []struct {
		name                 string
		valAddr              fmt.Stringer
		args                 []string
		expectErr            bool
		expectedCode         uint32
		respType             proto.Message
		expectedResponseType []string
	}{
		{
			"invalid validator address",
			val.Address,
			[]string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			true, 0, nil,
			[]string{},
		},
		{
			"valid transaction",
			sdk.ValAddress(val.Address),
			[]string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			false, 0, &sdk.TxResponse{},
			[]string{
				"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorRewardResponse",
			},
		},
		{
			"valid transaction (with commission)",
			sdk.ValAddress(val.Address),
			[]string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=true", cli.FlagCommission),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			false, 0, &sdk.TxResponse{},
			[]string{
				"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorRewardResponse",
				"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommissionResponse",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			clientCtx := val.ClientCtx

			_, _ = s.network.WaitForHeightWithTimeout(10, time.Minute)
			bz, err := MsgWithdrawDelegatorRewardExec(clientCtx, tc.valAddr, tc.args...)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(bz, tc.respType), string(bz))

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code)

				data, err := hex.DecodeString(txResp.Data)
				s.Require().NoError(err)

				txMsgData := sdk.TxMsgData{}
				err = s.cfg.Codec.Unmarshal(data, &txMsgData)
				s.Require().NoError(err)
				for responseIdx, msgResponse := range txMsgData.MsgResponses {
					s.Require().Equal(tc.expectedResponseType[responseIdx], msgResponse.TypeUrl)
					switch msgResponse.TypeUrl {
					case "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorRewardResponse":
						var resp distrtypes.MsgWithdrawDelegatorRewardResponse
						// can't use unpackAny as response types are not registered.
						err = s.cfg.Codec.Unmarshal(msgResponse.Value, &resp)
						s.Require().NoError(err)
						s.Require().True(resp.Amount.IsAllGT(sdk.NewCoins(sdk.NewCoin("stake", sdk.OneInt()))),
							fmt.Sprintf("expected a positive coin value, got %v", resp.Amount))
					case "/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommissionResponse":
						var resp distrtypes.MsgWithdrawValidatorCommissionResponse
						// can't use unpackAny as response types are not registered.
						err = s.cfg.Codec.Unmarshal(msgResponse.Value, &resp)
						s.Require().NoError(err)
						s.Require().True(resp.Amount.IsAllGT(sdk.NewCoins(sdk.NewCoin("stake", sdk.OneInt()))),
							fmt.Sprintf("expected a positive coin value, got %v", resp.Amount))
					}
				}
			}
		})
	}
}

func (s *IntegrationTestSuite) TestNewWithdrawAllRewardsCmd() {
	// reset the suite
	s.TearDownSuite()
	s.SetupSuite()

	val := s.network.Validators[0]

	testCases := []struct {
		name                 string
		args                 []string
		expectErr            bool
		expectedCode         uint32
		respType             proto.Message
		expectedResponseType []string
	}{
		{
			"valid transaction (offline)",
			[]string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagOffline),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			true, 0, nil,
			[]string{},
		},
		{
			"valid transaction",
			[]string{
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			false, 0, &sdk.TxResponse{},
			[]string{
				"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorRewardResponse",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.NewWithdrawAllRewardsCmd()
			clientCtx := val.ClientCtx

			_, _ = s.network.WaitForHeightWithTimeout(10, time.Minute)

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code)

				data, err := hex.DecodeString(txResp.Data)
				s.Require().NoError(err)

				txMsgData := sdk.TxMsgData{}
				err = s.cfg.Codec.Unmarshal(data, &txMsgData)
				s.Require().NoError(err)
				for responseIdx, msgResponse := range txMsgData.MsgResponses {
					s.Require().Equal(tc.expectedResponseType[responseIdx], msgResponse.TypeUrl)
					switch msgResponse.TypeUrl {
					case "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorRewardResponse":
						var resp distrtypes.MsgWithdrawDelegatorRewardResponse
						// can't use unpackAny as response types are not registered.
						err = s.cfg.Codec.Unmarshal(msgResponse.Value, &resp)
						s.Require().NoError(err)
						s.Require().True(resp.Amount.IsAllGT(sdk.NewCoins(sdk.NewCoin("stake", sdk.OneInt()))),
							fmt.Sprintf("expected a positive coin value, got %v", resp.Amount))
					case "/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommissionResponse":
						var resp distrtypes.MsgWithdrawValidatorCommissionResponse
						// can't use unpackAny as response types are not registered.
						err = s.cfg.Codec.Unmarshal(msgResponse.Value, &resp)
						s.Require().NoError(err)
						s.Require().True(resp.Amount.IsAllGT(sdk.NewCoins(sdk.NewCoin("stake", sdk.OneInt()))),
							fmt.Sprintf("expected a positive coin value, got %v", resp.Amount))
					}
				}
			}
		})
	}
}

func (s *IntegrationTestSuite) TestNewSetWithdrawAddrCmd() {
	// reset the suite
	s.TearDownSuite()
	s.SetupSuite()

	val := s.network.Validators[0]

	testCases := []struct {
		name         string
		args         []string
		expectErr    bool
		expectedCode uint32
		respType     proto.Message
	}{
		{
			"invalid withdraw address",
			[]string{
				"foo",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			true, 0, nil,
		},
		{
			"valid transaction",
			[]string{
				val.Address.String(),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			false, 0, &sdk.TxResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.NewSetWithdrawAddrCmd()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestNewFundCommunityPoolCmd() {
	// reset the suite
	s.TearDownSuite()
	s.SetupSuite()

	val := s.network.Validators[0]

	testCases := []struct {
		name         string
		args         []string
		expectErr    bool
		expectedCode uint32
		respType     proto.Message
	}{
		{
			"invalid funding amount",
			[]string{
				"-43foocoin",
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			true, 0, nil,
		},
		{
			"valid transaction",
			[]string{
				sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(5431))).String(),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			false, 0, &sdk.TxResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.NewFundCommunityPoolCmd()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestGetCmdSubmitProposal() {
	// reset the suite
	s.TearDownSuite()
	s.SetupSuite()

	val := s.network.Validators[0]
	invalidProp := `{
  "title": "",
  "description": "Pay me some Atoms!",
  "recipient": "foo",
  "amount": "-343foocoin",
  "deposit": -324foocoin
}`

	// fund some tokens to the community pool
	args := []string{
		sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(5431))).String(),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
	}

	invalidPropFile := testutil.WriteToNewTempFile(s.T(), invalidProp)
	cmd := cli.NewFundCommunityPoolCmd()
	out, err := clitestutil.ExecTestCLICmd(val.ClientCtx, cmd, args)
	s.Require().NoError(err)

	var txResp sdk.TxResponse
	s.Require().NoError(val.ClientCtx.Codec.UnmarshalJSON(out.Bytes(), &txResp), out.String())
	s.Require().Equal(uint32(0), txResp.Code)

	validProp := fmt.Sprintf(`{
  "title": "Community Pool Spend",
  "description": "Pay me some Atoms!",
  "recipient": "%s",
  "amount": "%s",
  "deposit": "%s"
}`, val.Address.String(), sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(5431)), sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(5431)))

	validPropFile := testutil.WriteToNewTempFile(s.T(), validProp)
	testCases := []struct {
		name         string
		args         []string
		expectErr    bool
		expectedCode uint32
		respType     proto.Message
	}{
		{
			"invalid proposal",
			[]string{
				invalidPropFile.Name(),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			true, 0, nil,
		},
		{
			"valid transaction",
			[]string{
				validPropFile.Name(),
				fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			},
			false, 0, &sdk.TxResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc

		s.Run(tc.name, func() {
			cmd := cli.GetCmdSubmitProposal()
			clientCtx := val.ClientCtx
			flags.AddTxFlagsToCmd(cmd)

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, tc.args)
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())
			}
		})
	}
}

func (s *IntegrationTestSuite) TestNewChangeBaseAddressCmd() {
	// reset the suite
	s.TearDownSuite()
	s.SetupSuite()

	val := s.network.Validators[0]

	testCases := []struct {
		name         string
		sender       string
		expectErr    bool
		expectedCode uint32
		respType     proto.Message
	}{
		{
			"wrong moderator",
			val.Address.String(),
			true, 0, nil,
		},
		{
			"correct moderator",
			distModeratorAddr,
			false, 0, &sdk.TxResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		args := []string{
			changeAddr,
			fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
			fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
			fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
		}

		s.Run(tc.name, func() {
			cmd := cli.NewChangeBaseAddressCmd()
			clientCtx := val.ClientCtx

			args = append(args, fmt.Sprintf("--%s=%s", flags.FlagFrom, tc.sender))
			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, args)
			if tc.expectErr {
				s.Require().Contains(out.String(), distrtypes.ErrInvalidModerator.Error())
			} else {
				s.Require().NoError(err)
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())
			}
		})
	}
}

func (s *IntegrationTestSuite) TestNewChangeModeratorCmd() {
	// reset the suite
	s.TearDownSuite()
	s.SetupSuite()

	val := s.network.Validators[0]

	testCases := []struct {
		name         string
		sender       string
		expectErr    bool
		expectedCode uint32
		respType     proto.Message
	}{
		{
			"wrong moderator",
			val.Address.String(),
			true, 0, nil,
		},
		{
			"correct moderator",
			distModeratorAddr,
			false, 0, &sdk.TxResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		args := []string{
			changeAddr,
			fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
			fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
			fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
		}

		s.Run(tc.name, func() {
			cmd := cli.NewChangeModeratorCmd()
			clientCtx := val.ClientCtx

			args = append(args, fmt.Sprintf("--%s=%s", flags.FlagFrom, tc.sender))
			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, args)
			if tc.expectErr {
				s.Require().Contains(out.String(), distrtypes.ErrInvalidModerator.Error())
			} else {
				s.Require().NoError(err)
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())
			}
		})
	}
}

func (s *IntegrationTestSuite) TestNewChangeRatioCmd() {
	// reset the suite
	s.TearDownSuite()
	s.SetupSuite()

	val := s.network.Validators[0]

	testCases := []struct {
		name          string
		sender        string
		newRatio      []string
		expectErr     bool
		expectedCode  uint32
		expectedError string
		respType      proto.Message
	}{
		{
			"wrong moderator",
			val.Address.String(),
			[]string{"0.34", "0.33", "0.33"},
			true, 0,
			distrtypes.ErrInvalidModerator.Error(),
			nil,
		},
		{
			"correct moderator wrong ratio",
			distModeratorAddr,
			[]string{"0.34", "0.33", "0.32"},
			true, 0,
			distrtypes.ErrInvalidRatio.Error(),
			nil,
		},
		{
			"correct moderator correct ratio",
			distModeratorAddr,
			[]string{"0.34", "0.33", "0.33"},
			false, 0, "", &sdk.TxResponse{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		args := []string{
			tc.newRatio[0], tc.newRatio[1], tc.newRatio[2],
			fmt.Sprintf("--%s=%s", flags.FlagFrom, tc.sender),
			fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
			fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
			fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
		}

		s.Run(tc.name, func() {
			cmd := cli.NewChangeRatioCmd()
			clientCtx := val.ClientCtx

			out, err := clitestutil.ExecTestCLICmd(clientCtx, cmd, args)
			if tc.expectErr {
				s.Require().Contains(out.String(), tc.expectedError)
			} else {
				s.Require().NoError(err)
				s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), tc.respType), out.String())

				txResp := tc.respType.(*sdk.TxResponse)
				s.Require().Equal(tc.expectedCode, txResp.Code, out.String())
			}
		})
	}
}

func (s *IntegrationTestSuite) TestFeeDistribution() {
	// reset the suite
	s.TearDownSuite()
	s.SetupSuite()

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

		argsTx := []string{
			val.Address.String(), changeAddr, "1000" + s.network.Config.BondDenom,
			fmt.Sprintf("--%s=%s", flags.FlagFrom, val.Address.String()),
			fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
			fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
			fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(300))).String()),
		}

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
		// fee: 300stake || increased supply for 5 last blocks: 265stake || burn: 100stake
		s.Require().Equal(oldTS.Supply.String(),
			newTS.Supply.
				Sub(sdk.NewCoin(s.network.Config.BondDenom, sdk.NewInt(265))).
				Add(sdk.NewCoin(s.network.Config.BondDenom, sdk.NewInt(100))).String())

		// test 1/3 base address fee
		// reward for last 5 blocks: 130stake || bank send tx fee rewards: 100stake
		s.Require().Equal(oldBaseBalance.Balances.String(),
			newBaseBalance.Balances.Sub(sdk.NewCoin(s.network.Config.BondDenom, sdk.NewInt(130))).
				Sub(sdk.NewCoin(s.network.Config.BondDenom, sdk.NewInt(100))).String())

		// test 1/3 staking rewards

		// community pool:
		// for last 5 blocks: 2.7stake || for the tx: 2stake
		s.Require().Equal(oldCommPool.Pool.String(),
			newCommPool.Pool.Sub(sdk.NewDecCoins(sdk.NewDecCoinFromDec(s.network.Config.BondDenom, sdk.NewDecWithPrec(27, 1)))).
				Sub(sdk.NewDecCoins(sdk.NewDecCoin(s.network.Config.BondDenom, sdk.NewInt(2)))).String())

		// delegator rewards
		// for last 5 blocks: 66.15stake || for the tx: 49stake
		s.Require().Equal(oldRewards.Rewards.String(),
			newRewards.Rewards.Sub(sdk.NewDecCoins(sdk.NewDecCoinFromDec(s.network.Config.BondDenom, sdk.NewDecWithPrec(6615, 2)))).
				Sub(sdk.NewDecCoins(sdk.NewDecCoin(s.network.Config.BondDenom, sdk.NewInt(49)))).String())

		// validator commission
		// for last 5 blocks: 66.15stake || for the tx: 49stake
		s.Require().Equal(oldValCommission.Commission.String(),
			newValCommission.Commission.Sub(sdk.NewDecCoins(sdk.NewDecCoinFromDec(s.network.Config.BondDenom, sdk.NewDecWithPrec(6615, 2)))).
				Sub(sdk.NewDecCoins(sdk.NewDecCoin(s.network.Config.BondDenom, sdk.NewInt(49)))).String())

	})
}

func (s *IntegrationTestSuite) addSignerKey(uid, addr, mnic string) {
	signerAcc, _ := s.createAccount(uid, mnic)
	s.Require().Equal(signerAcc.String(), addr)
}

func (s *IntegrationTestSuite) createAccount(uid, mnemonic string) (sdk.AccAddress, string) {
	kb := s.network.Validators[0].ClientCtx.Keyring
	keyringAlgos, _ := kb.SupportedAlgorithms()
	algo, err := keyring.NewSigningAlgoFromString(s.network.Config.SigningAlgo, keyringAlgos)
	s.Require().NoError(err)

	account, secret, err := testutil.GenerateSaveCoinKey(kb, uid, mnemonic, true, algo)
	s.Require().NoError(err)

	return account, secret
}
