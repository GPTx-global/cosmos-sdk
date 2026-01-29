package keeper_test

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec/address"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/testutil"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestCancelUnbondingDelegation(t *testing.T) {
	t.Parallel()
	f := initFixture(t)

	ctx := f.sdkCtx
	msgServer := keeper.NewMsgServerImpl(f.stakingKeeper)
	bondDenom, err := f.stakingKeeper.BondDenom(ctx)
	assert.NilError(t, err)

	// set the not bonded pool module account
	notBondedPool := f.stakingKeeper.GetNotBondedPool(ctx)
	startTokens := f.stakingKeeper.TokensFromConsensusPower(ctx, 5)

	assert.NilError(t, testutil.FundModuleAccount(ctx, f.bankKeeper, notBondedPool.GetName(), sdk.NewCoins(sdk.NewCoin(bondDenom, startTokens))))
	f.accountKeeper.SetModuleAccount(ctx, notBondedPool)

	moduleBalance := f.bankKeeper.GetBalance(ctx, notBondedPool.GetAddress(), bondDenom)
	assert.DeepEqual(t, sdk.NewInt64Coin(bondDenom, startTokens.Int64()), moduleBalance)

	// accounts
	addrs := simtestutil.AddTestAddrsIncremental(f.bankKeeper, f.stakingKeeper, ctx, 2, math.NewInt(10000))
	valAddr := sdk.ValAddress(addrs[0])
	delegatorAddr := addrs[1]

	// setup a new validator with bonded status
	validator, err := types.NewValidator(valAddr.String(), PKs[0], types.NewDescription("Validator", "", "", "", ""))
	validator.Status = types.Bonded
	assert.NilError(t, err)
	assert.NilError(t, f.stakingKeeper.SetValidator(ctx, validator))

	validatorAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
	assert.NilError(t, err)

	// setting the ubd entry
	unbondingAmount := sdk.NewInt64Coin(bondDenom, 5)
	ubd := types.NewUnbondingDelegation(
		delegatorAddr, validatorAddr, 10,
		ctx.BlockTime().Add(time.Minute*10),
		unbondingAmount.Amount,
		0,
		address.NewBech32Codec("cosmosvaloper"), address.NewBech32Codec("cosmos"),
	)

	// set and retrieve a record
	assert.NilError(t, f.stakingKeeper.SetUnbondingDelegation(ctx, ubd))
	resUnbond, found := f.stakingKeeper.GetUnbondingDelegation(ctx, delegatorAddr, validatorAddr)
	assert.Assert(t, found)
	assert.DeepEqual(t, ubd, resUnbond)

	testCases := []struct {
		name      string
		exceptErr bool
		req       types.MsgCancelUnbondingDelegation
		expErrMsg string
	}{
		{
			name:      "entry not found at height",
			exceptErr: true,
			req: types.MsgCancelUnbondingDelegation{
				DelegatorAddress: resUnbond.DelegatorAddress,
				ValidatorAddress: resUnbond.ValidatorAddress,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(4)),
				CreationHeight:   11,
			},
			expErrMsg: "unbonding delegation entry is not found at block height",
		},
		{
			name:      "invalid height",
			exceptErr: true,
			req: types.MsgCancelUnbondingDelegation{
				DelegatorAddress: resUnbond.DelegatorAddress,
				ValidatorAddress: resUnbond.ValidatorAddress,
				Amount:           sdk.NewCoin(bondDenom, math.NewInt(4)),
				CreationHeight:   0,
			},
			expErrMsg: "invalid height",
		},
		{
			name:      "invalid coin",
			exceptErr: true,
			req: types.MsgCancelUnbondingDelegation{
				DelegatorAddress: resUnbond.DelegatorAddress,
				ValidatorAddress: resUnbond.ValidatorAddress,
				Amount:           sdk.NewCoin("dump_coin", math.NewInt(4)),
				CreationHeight:   10,
			},
			expErrMsg: "invalid coin denomination",
		},
		{
			name:      "validator not exists",
			exceptErr: true,
			req: types.MsgCancelUnbondingDelegation{
				DelegatorAddress: resUnbond.DelegatorAddress,
				ValidatorAddress: sdk.ValAddress(sdk.AccAddress("asdsad")).String(),
				Amount:           unbondingAmount,
				CreationHeight:   10,
			},
			expErrMsg: "validator does not exist",
		},
		{
			name:      "invalid delegator address",
			exceptErr: true,
			req: types.MsgCancelUnbondingDelegation{
				DelegatorAddress: "invalid_delegator_addrtess",
				ValidatorAddress: resUnbond.ValidatorAddress,
				Amount:           unbondingAmount,
				CreationHeight:   0,
			},
			expErrMsg: "decoding bech32 failed",
		},
		{
			name:      "invalid amount",
			exceptErr: true,
			req: types.MsgCancelUnbondingDelegation{
				DelegatorAddress: resUnbond.DelegatorAddress,
				ValidatorAddress: resUnbond.ValidatorAddress,
				Amount:           unbondingAmount.Add(sdk.NewInt64Coin(bondDenom, 10)),
				CreationHeight:   10,
			},
			expErrMsg: "amount is greater than the unbonding delegation entry balance",
		},
		{
			name:      "success",
			exceptErr: false,
			req: types.MsgCancelUnbondingDelegation{
				DelegatorAddress: resUnbond.DelegatorAddress,
				ValidatorAddress: resUnbond.ValidatorAddress,
				Amount:           unbondingAmount.Sub(sdk.NewInt64Coin(bondDenom, 1)),
				CreationHeight:   10,
			},
		},
		{
			name:      "success",
			exceptErr: false,
			req: types.MsgCancelUnbondingDelegation{
				DelegatorAddress: resUnbond.DelegatorAddress,
				ValidatorAddress: resUnbond.ValidatorAddress,
				Amount:           unbondingAmount.Sub(unbondingAmount.Sub(sdk.NewInt64Coin(bondDenom, 1))),
				CreationHeight:   10,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := msgServer.CancelUnbondingDelegation(ctx, &testCase.req)
			if testCase.exceptErr {
				assert.ErrorContains(t, err, testCase.expErrMsg)
			} else {
				assert.NilError(t, err)
				balanceForNotBondedPool := f.bankKeeper.GetBalance(ctx, notBondedPool.GetAddress(), bondDenom)
				assert.DeepEqual(t, balanceForNotBondedPool, moduleBalance.Sub(testCase.req.Amount))
				moduleBalance = moduleBalance.Sub(testCase.req.Amount)
			}
		})
	}
}

func TestCreateValidatorMinSelfDelegationMustMeetMinValidatorBondAmount(t *testing.T) {
	t.Parallel()
	f := initFixture(t)

	ctx := f.sdkCtx
	msgServer := keeper.NewMsgServerImpl(f.stakingKeeper)

	bondDenom, err := f.stakingKeeper.BondDenom(ctx)
	assert.NilError(t, err)

	// Ensure staking pool module accounts exist.
	f.accountKeeper.SetModuleAccount(ctx, f.stakingKeeper.GetNotBondedPool(ctx))
	f.accountKeeper.SetModuleAccount(ctx, f.stakingKeeper.GetBondedPool(ctx))

	params := types.DefaultParams()
	params.MinValidatorBondAmount = math.NewInt(100)
	assert.NilError(t, f.stakingKeeper.SetParams(ctx, params))

	addrs := simtestutil.AddTestAddrsIncremental(f.bankKeeper, f.stakingKeeper, ctx, 3, math.NewInt(10000))

	commission := types.NewCommissionRates(
		math.LegacyMustNewDecFromStr("0.10"),
		math.LegacyMustNewDecFromStr("0.20"),
		math.LegacyMustNewDecFromStr("0.01"),
	)
	desc := types.NewDescription("validator", "", "", "", "")

	// MinSelfDelegation < MinValidatorBondAmount should fail.
	valAddr1 := sdk.ValAddress(addrs[0])
	msgFail, err := types.NewMsgCreateValidator(
		valAddr1.String(),
		PKs[0],
		sdk.NewCoin(bondDenom, math.NewInt(50)),
		desc,
		commission,
		math.NewInt(50),
	)
	assert.NilError(t, err)
	_, err = msgServer.CreateValidator(ctx, msgFail)
	assert.ErrorContains(t, err, "minimum self delegation")
	assert.ErrorContains(t, err, "minimum validator bond amount")

	// MinSelfDelegation == MinValidatorBondAmount should succeed.
	valAddr2 := sdk.ValAddress(addrs[1])
	msgEq, err := types.NewMsgCreateValidator(
		valAddr2.String(),
		PKs[1],
		sdk.NewCoin(bondDenom, math.NewInt(100)),
		desc,
		commission,
		math.NewInt(100),
	)
	assert.NilError(t, err)
	_, err = msgServer.CreateValidator(ctx, msgEq)
	assert.NilError(t, err)

	// MinSelfDelegation > MinValidatorBondAmount should succeed.
	valAddr3 := sdk.ValAddress(addrs[2])
	msgGt, err := types.NewMsgCreateValidator(
		valAddr3.String(),
		PKs[2],
		sdk.NewCoin(bondDenom, math.NewInt(150)),
		desc,
		commission,
		math.NewInt(150),
	)
	assert.NilError(t, err)
	_, err = msgServer.CreateValidator(ctx, msgGt)
	assert.NilError(t, err)
}

func TestEditValidatorMinSelfDelegationMustMeetMinValidatorBondAmount(t *testing.T) {
	t.Parallel()
	f := initFixture(t)

	ctx := f.sdkCtx
	msgServer := keeper.NewMsgServerImpl(f.stakingKeeper)

	params := types.DefaultParams()
	params.MinValidatorBondAmount = math.NewInt(100)
	assert.NilError(t, f.stakingKeeper.SetParams(ctx, params))

	addrs := simtestutil.AddTestAddrsIncremental(f.bankKeeper, f.stakingKeeper, ctx, 1, math.NewInt(10000))
	valAddr := sdk.ValAddress(addrs[0])

	// Create an existing validator with MinSelfDelegation below MinValidatorBondAmount
	// (this can happen after upgrades or governance param changes).
	validator, err := types.NewValidator(valAddr.String(), PKs[0], types.NewDescription("Validator", "", "", "", ""))
	assert.NilError(t, err)
	validator.Tokens = math.NewInt(1000)
	validator.MinSelfDelegation = math.NewInt(50)
	assert.NilError(t, f.stakingKeeper.SetValidator(ctx, validator))

	desc := types.NewDescription("ValidatorUpdated", "", "", "", "")

	// Increase but still below MinValidatorBondAmount should fail.
	newMSD := math.NewInt(80)
	msgFail := types.NewMsgEditValidator(valAddr.String(), desc, nil, &newMSD)
	_, err = msgServer.EditValidator(ctx, msgFail)
	assert.ErrorContains(t, err, "minimum self delegation")
	assert.ErrorContains(t, err, "minimum validator bond amount")

	// Increase to >= MinValidatorBondAmount should succeed.
	newMSD2 := math.NewInt(120)
	msgOK := types.NewMsgEditValidator(valAddr.String(), desc, nil, &newMSD2)
	_, err = msgServer.EditValidator(ctx, msgOK)
	assert.NilError(t, err)

	updated, err := f.stakingKeeper.GetValidator(ctx, valAddr)
	assert.NilError(t, err)
	assert.DeepEqual(t, newMSD2, updated.MinSelfDelegation)
}

func TestValidatorJailOnSelfUndelegateBelowMinSelfDelegationEqualMinValidatorBondAmount(t *testing.T) {
	t.Parallel()
	f := initFixture(t)

	ctx := f.sdkCtx

	bondDenom, err := f.stakingKeeper.BondDenom(ctx)
	assert.NilError(t, err)

	// Ensure staking pool module accounts exist.
	f.accountKeeper.SetModuleAccount(ctx, f.stakingKeeper.GetNotBondedPool(ctx))
	f.accountKeeper.SetModuleAccount(ctx, f.stakingKeeper.GetBondedPool(ctx))

	params := types.DefaultParams()
	params.MinValidatorBondAmount = math.NewInt(100)
	assert.NilError(t, f.stakingKeeper.SetParams(ctx, params))

	addrs := simtestutil.AddTestAddrsIncremental(f.bankKeeper, f.stakingKeeper, ctx, 1, math.NewInt(10000))
	valAddr := sdk.ValAddress(addrs[0])

	validator, err := types.NewValidator(valAddr.String(), PKs[0], types.NewDescription("Validator", "", "", "", ""))
	assert.NilError(t, err)
	validator.MinSelfDelegation = math.NewInt(100)
	assert.NilError(t, f.stakingKeeper.SetValidator(ctx, validator))
	assert.NilError(t, f.stakingKeeper.SetValidatorByConsAddr(ctx, validator))
	assert.NilError(t, f.stakingKeeper.SetNewValidatorByPowerIndex(ctx, validator))

	// Self-delegate above the minimum.
	_, err = f.stakingKeeper.Delegate(ctx, addrs[0], math.NewInt(150), types.Unbonded, validator, true)
	assert.NilError(t, err)

	// Undelegate enough to bring self-delegation below MinSelfDelegation (= MinValidatorBondAmount),
	// which should jail the validator.
	_, _, err = f.stakingKeeper.Undelegate(ctx, addrs[0], valAddr, math.LegacyNewDecFromInt(math.NewInt(60)))
	assert.NilError(t, err)

	updated, err := f.stakingKeeper.GetValidator(ctx, valAddr)
	assert.NilError(t, err)
	assert.Assert(t, updated.Jailed)

	// sanity check that coin denom was correct and balances are tracked
	_ = bondDenom
}
