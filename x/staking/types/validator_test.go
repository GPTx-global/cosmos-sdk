package types_test

import (
	"math/rand"
	"sort"
	"testing"

	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/testutil"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestValidatorTestEquivalent(t *testing.T) {
	val1 := newValidator(t, valAddr1, pk1)
	val2 := newValidator(t, valAddr1, pk1)
	require.Equal(t, val1.String(), val2.String())

	val2 = newValidator(t, valAddr2, pk2)
	require.NotEqual(t, val1.String(), val2.String())
}

func TestUpdateDescription(t *testing.T) {
	d1 := types.Description{
		Website: "https://validator.cosmos",
		Details: "Test validator",
	}

	d2 := types.Description{
		Moniker:  types.DoNotModifyDesc,
		Identity: types.DoNotModifyDesc,
		Website:  types.DoNotModifyDesc,
		Details:  types.DoNotModifyDesc,
	}

	d3 := types.Description{
		Moniker:  "",
		Identity: "",
		Website:  "",
		Details:  "",
	}

	d, err := d1.UpdateDescription(d2)
	require.Nil(t, err)
	require.Equal(t, d, d1)

	d, err = d1.UpdateDescription(d3)
	require.Nil(t, err)
	require.Equal(t, d, d3)
}

func TestABCIValidatorUpdate(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	abciVal := validator.ABCIValidatorUpdate(sdk.DefaultPowerReduction)
	pk, err := validator.TmConsPublicKey()
	require.NoError(t, err)
	require.Equal(t, pk, abciVal.PubKey)
	require.Equal(t, validator.BondedTokens().Int64(), abciVal.Power)
}

func TestABCIValidatorUpdateZero(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	abciVal := validator.ABCIValidatorUpdateZero()
	pk, err := validator.TmConsPublicKey()
	require.NoError(t, err)
	require.Equal(t, pk, abciVal.PubKey)
	require.Equal(t, int64(0), abciVal.Power)
}

func TestShareTokens(t *testing.T) {
	validator := mkValidator(100, math.LegacyNewDec(100))
	assert.True(math.LegacyDecEq(t, math.LegacyNewDec(50), validator.TokensFromShares(math.LegacyNewDec(50))))

	validator.Tokens = math.NewInt(50)
	assert.True(math.LegacyDecEq(t, math.LegacyNewDec(25), validator.TokensFromShares(math.LegacyNewDec(50))))
	assert.True(math.LegacyDecEq(t, math.LegacyNewDec(5), validator.TokensFromShares(math.LegacyNewDec(10))))
}

func TestRemoveTokens(t *testing.T) {
	validator := mkValidator(100, math.LegacyNewDec(100))

	// remove tokens and test check everything
	validator = validator.RemoveTokens(math.NewInt(10))
	require.Equal(t, int64(90), validator.Tokens.Int64())

	// update validator to from bonded -> unbonded
	validator = validator.UpdateStatus(types.Unbonded)
	require.Equal(t, types.Unbonded, validator.Status)

	validator = validator.RemoveTokens(math.NewInt(10))
	require.Panics(t, func() { validator.RemoveTokens(math.NewInt(-1)) })
	require.Panics(t, func() { validator.RemoveTokens(math.NewInt(100)) })
}

func TestAddTokensValidatorBonded(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	validator = validator.UpdateStatus(types.Bonded)
	validator, delShares := validator.AddTokensFromDel(math.NewInt(10))

	assert.True(math.LegacyDecEq(t, math.LegacyNewDec(10), delShares))
	assert.True(math.IntEq(t, math.NewInt(10), validator.BondedTokens()))
	assert.True(math.LegacyDecEq(t, math.LegacyNewDec(10), validator.DelegatorShares))
}

func TestAddTokensValidatorUnbonding(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	validator = validator.UpdateStatus(types.Unbonding)
	validator, delShares := validator.AddTokensFromDel(math.NewInt(10))

	assert.True(math.LegacyDecEq(t, math.LegacyNewDec(10), delShares))
	assert.Equal(t, types.Unbonding, validator.Status)
	assert.True(math.IntEq(t, math.NewInt(10), validator.Tokens))
	assert.True(math.LegacyDecEq(t, math.LegacyNewDec(10), validator.DelegatorShares))
}

func TestAddTokensValidatorUnbonded(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	validator = validator.UpdateStatus(types.Unbonded)
	validator, delShares := validator.AddTokensFromDel(math.NewInt(10))

	assert.True(math.LegacyDecEq(t, math.LegacyNewDec(10), delShares))
	assert.Equal(t, types.Unbonded, validator.Status)
	assert.True(math.IntEq(t, math.NewInt(10), validator.Tokens))
	assert.True(math.LegacyDecEq(t, math.LegacyNewDec(10), validator.DelegatorShares))
}

// TODO refactor to make simpler like the AddToken tests above
func TestRemoveDelShares(t *testing.T) {
	valA := types.Validator{
		OperatorAddress: valAddr1.String(),
		ConsensusPubkey: pk1Any,
		Status:          types.Bonded,
		Tokens:          math.NewInt(100),
		DelegatorShares: math.LegacyNewDec(100),
	}

	// Remove delegator shares
	valB, coinsB := valA.RemoveDelShares(math.LegacyNewDec(10))
	require.Equal(t, int64(10), coinsB.Int64())
	require.Equal(t, int64(90), valB.DelegatorShares.RoundInt64())
	require.Equal(t, int64(90), valB.BondedTokens().Int64())

	// specific case from random tests
	validator := mkValidator(5102, math.LegacyNewDec(115))
	_, tokens := validator.RemoveDelShares(math.LegacyNewDec(29))

	require.True(math.IntEq(t, math.NewInt(1286), tokens))
}

func TestAddTokensFromDel(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)

	validator, shares := validator.AddTokensFromDel(math.NewInt(6))
	require.True(math.LegacyDecEq(t, math.LegacyNewDec(6), shares))
	require.True(math.LegacyDecEq(t, math.LegacyNewDec(6), validator.DelegatorShares))
	require.True(math.IntEq(t, math.NewInt(6), validator.Tokens))

	validator, shares = validator.AddTokensFromDel(math.NewInt(3))
	require.True(math.LegacyDecEq(t, math.LegacyNewDec(3), shares))
	require.True(math.LegacyDecEq(t, math.LegacyNewDec(9), validator.DelegatorShares))
	require.True(math.IntEq(t, math.NewInt(9), validator.Tokens))
}

func TestUpdateStatus(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	validator, _ = validator.AddTokensFromDel(math.NewInt(100))
	require.Equal(t, types.Unbonded, validator.Status)
	require.Equal(t, int64(100), validator.Tokens.Int64())

	// Unbonded to Bonded
	validator = validator.UpdateStatus(types.Bonded)
	require.Equal(t, types.Bonded, validator.Status)

	// Bonded to Unbonding
	validator = validator.UpdateStatus(types.Unbonding)
	require.Equal(t, types.Unbonding, validator.Status)

	// Unbonding to Bonded
	validator = validator.UpdateStatus(types.Bonded)
	require.Equal(t, types.Bonded, validator.Status)
}

func TestPossibleOverflow(t *testing.T) {
	delShares := math.LegacyNewDec(391432570689183511).Quo(math.LegacyNewDec(40113011844664))
	validator := mkValidator(2159, delShares)
	newValidator, _ := validator.AddTokensFromDel(math.NewInt(71))

	require.False(t, newValidator.DelegatorShares.IsNegative())
	require.False(t, newValidator.Tokens.IsNegative())
}

func TestValidatorMarshalUnmarshalJSON(t *testing.T) {
	validator := newValidator(t, valAddr1, pk1)
	js, err := legacy.Cdc.MarshalJSON(validator)
	require.NoError(t, err)
	require.NotEmpty(t, js)
	require.Contains(t, string(js), "\"consensus_pubkey\":{\"type\":\"tendermint/PubKeyEd25519\"")
	got := &types.Validator{}
	err = legacy.Cdc.UnmarshalJSON(js, got)
	assert.NoError(t, err)
	assert.True(t, validator.Equal(got))
}

func TestValidatorSetInitialCommission(t *testing.T) {
	val := newValidator(t, valAddr1, pk1)
	testCases := []struct {
		validator   types.Validator
		commission  types.Commission
		expectedErr bool
	}{
		{val, types.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()), false},
		{val, types.NewCommission(math.LegacyZeroDec(), math.LegacyNewDecWithPrec(-1, 1), math.LegacyZeroDec()), true},
		{val, types.NewCommission(math.LegacyZeroDec(), math.LegacyNewDec(15000000000), math.LegacyZeroDec()), true},
		{val, types.NewCommission(math.LegacyNewDecWithPrec(-1, 1), math.LegacyZeroDec(), math.LegacyZeroDec()), true},
		{val, types.NewCommission(math.LegacyNewDecWithPrec(2, 1), math.LegacyNewDecWithPrec(1, 1), math.LegacyZeroDec()), true},
		{val, types.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyNewDecWithPrec(-1, 1)), true},
		{val, types.NewCommission(math.LegacyZeroDec(), math.LegacyNewDecWithPrec(1, 1), math.LegacyNewDecWithPrec(2, 1)), true},
	}

	for i, tc := range testCases {
		val, err := tc.validator.SetInitialCommission(tc.commission)

		if tc.expectedErr {
			require.Error(t, err,
				"expected error for test case #%d with commission: %s", i, tc.commission,
			)
		} else {
			require.NoError(t, err,
				"unexpected error for test case #%d with commission: %s", i, tc.commission,
			)
			require.Equal(t, tc.commission, val.Commission,
				"invalid validator commission for test case #%d with commission: %s", i, tc.commission,
			)
		}
	}
}

// Check that sort will create deterministic ordering of validators
func TestValidatorsSortDeterminism(t *testing.T) {
	vals := make([]types.Validator, 10)
	sortedVals := make([]types.Validator, 10)

	// Create random validator slice
	for i := range vals {
		pk := ed25519.GenPrivKey().PubKey()
		vals[i] = newValidator(t, sdk.ValAddress(pk.Address()), pk)
	}

	// Save sorted copy
	sort.Sort(types.Validators{Validators: vals, ValidatorCodec: address.NewBech32Codec("cosmosvaloper")})
	copy(sortedVals, vals)

	// Randomly shuffle validators, sort, and check it is equal to original sort
	for range 10 {
		rand.Shuffle(10, func(i, j int) {
			vals[i], vals[j] = vals[j], vals[i]
		})

		types.Validators{Validators: vals, ValidatorCodec: address.NewBech32Codec("cosmosvaloper")}.Sort()
		require.Equal(t, sortedVals, vals, "Validator sort returned different slices")
	}
}

// Check SortCometBFT sorts the same as CometBFT
func TestValidatorsSortCometBFT(t *testing.T) {
	vals := make([]types.Validator, 100)

	for i := range vals {
		pk := ed25519.GenPrivKey().PubKey()
		pk2 := ed25519.GenPrivKey().PubKey()
		vals[i] = newValidator(t, sdk.ValAddress(pk2.Address()), pk)
		vals[i].Status = types.Bonded
		vals[i].Tokens = math.NewInt(rand.Int63())
	}
	// create some validators with the same power
	for i := range 10 {
		vals[i].Tokens = math.NewInt(1000000)
	}

	valz := types.Validators{Validators: vals, ValidatorCodec: address.NewBech32Codec("cosmosvaloper")}

	// create expected CometBFT validators by converting to CometBFT then sorting
	expectedVals, err := testutil.ToCmtValidators(valz, sdk.DefaultPowerReduction)
	require.NoError(t, err)
	sort.Sort(cmttypes.ValidatorsByVotingPower(expectedVals))

	// sort in SDK and then convert to CometBFT
	sort.SliceStable(valz.Validators, func(i, j int) bool {
		return types.ValidatorsByVotingPower(valz.Validators).Less(i, j, sdk.DefaultPowerReduction)
	})
	actualVals, err := testutil.ToCmtValidators(valz, sdk.DefaultPowerReduction)
	require.NoError(t, err)

	require.Equal(t, expectedVals, actualVals, "sorting in SDK is not the same as sorting in CometBFT")
}

func TestValidatorToCmt(t *testing.T) {
	vals := types.Validators{}
	expected := make([]*cmttypes.Validator, 10)

	for i := range 10 {
		pk := ed25519.GenPrivKey().PubKey()
		val := newValidator(t, sdk.ValAddress(pk.Address()), pk)
		val.Status = types.Bonded
		val.Tokens = math.NewInt(rand.Int63())
		vals.Validators = append(vals.Validators, val)
		cmtPk, err := cryptocodec.ToCmtPubKeyInterface(pk)
		require.NoError(t, err)
		expected[i] = cmttypes.NewValidator(cmtPk, val.ConsensusPower(sdk.DefaultPowerReduction))
	}
	vs, err := testutil.ToCmtValidators(vals, sdk.DefaultPowerReduction)
	require.NoError(t, err)
	require.Equal(t, expected, vs)
}

func TestBondStatus(t *testing.T) {
	require.False(t, types.Unbonded == types.Bonded)
	require.False(t, types.Unbonded == types.Unbonding)
	require.False(t, types.Bonded == types.Unbonding)
	require.Equal(t, types.BondStatus(4).String(), "4")
	require.Equal(t, types.BondStatusUnspecified, types.Unspecified.String())
	require.Equal(t, types.BondStatusUnbonded, types.Unbonded.String())
	require.Equal(t, types.BondStatusBonded, types.Bonded.String())
	require.Equal(t, types.BondStatusUnbonding, types.Unbonding.String())
}

func mkValidator(tokens int64, shares math.LegacyDec) types.Validator {
	return types.Validator{
		OperatorAddress: valAddr1.String(),
		ConsensusPubkey: pk1Any,
		Status:          types.Bonded,
		Tokens:          math.NewInt(tokens),
		DelegatorShares: shares,
	}
}

// Creates a new validators and asserts the error check.
func newValidator(t *testing.T, operator sdk.ValAddress, pubKey cryptotypes.PubKey) types.Validator {
	t.Helper()
	v, err := types.NewValidator(operator.String(), pubKey, types.Description{})
	require.NoError(t, err)
	return v
}

func TestValidatorMinSelfDelegation(t *testing.T) {
	val := newValidator(t, valAddr1, pk1)
	
	// Test default minSelfDelegation (should be one)
	require.Equal(t, math.OneInt(), val.MinSelfDelegation)
	
	// Test setting minSelfDelegation
	val.MinSelfDelegation = math.NewInt(100)
	require.Equal(t, int64(100), val.MinSelfDelegation.Int64())
	
	// Test that minSelfDelegation must be positive
	val.MinSelfDelegation = math.NewInt(0)
	require.True(t, val.MinSelfDelegation.GTE(math.ZeroInt()))
}

func TestValidatorMinValidatorBondAmount(t *testing.T) {
	// Test default min validator bond amount
	params := types.DefaultParams()
	require.Equal(t, int64(types.DefaultMinValidatorBondAmount), params.MinValidatorBondAmount.Int64())
	
	// Test creating params with custom min validator bond amount
	minBondAmount := math.NewInt(1000000)
	customParams := types.NewParams(
		types.DefaultUnbondingTime,
		types.DefaultMaxValidators,
		types.DefaultMaxEntries,
		types.DefaultHistoricalEntries,
		sdk.DefaultBondDenom,
		types.DefaultMinCommissionRate,
		minBondAmount,
	)
	require.Equal(t, minBondAmount, customParams.MinValidatorBondAmount)
	
	// Test validation: min validator bond amount cannot be negative
	invalidParams := types.Params{
		UnbondingTime:          types.DefaultUnbondingTime,
		MaxValidators:          types.DefaultMaxValidators,
		MaxEntries:             types.DefaultMaxEntries,
		HistoricalEntries:      types.DefaultHistoricalEntries,
		BondDenom:              sdk.DefaultBondDenom,
		MinCommissionRate:      types.DefaultMinCommissionRate,
		MinValidatorBondAmount: math.NewInt(-100),
	}
	err := invalidParams.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "minimum validator bond amount cannot be negative")
	
	// Test validation: valid min validator bond amount
	validParams := types.Params{
		UnbondingTime:          types.DefaultUnbondingTime,
		MaxValidators:          types.DefaultMaxValidators,
		MaxEntries:             types.DefaultMaxEntries,
		HistoricalEntries:      types.DefaultHistoricalEntries,
		BondDenom:              sdk.DefaultBondDenom,
		MinCommissionRate:      types.DefaultMinCommissionRate,
		MinValidatorBondAmount: math.NewInt(1000),
	}
	err = validParams.Validate()
	require.NoError(t, err)
}

func TestValidatorMinSelfDelegationWithMinValidatorBondAmount(t *testing.T) {
	val := newValidator(t, valAddr1, pk1)
	
	// Test case 1: minSelfDelegation should be at least minValidatorBondAmount
	minValidatorBondAmount := math.NewInt(1000000)
	minSelfDelegation := math.NewInt(500000)
	
	// This should fail validation as minSelfDelegation < minValidatorBondAmount
	require.True(t, minSelfDelegation.LT(minValidatorBondAmount))
	
	// Test case 2: minSelfDelegation equals minValidatorBondAmount (should pass)
	minSelfDelegation = math.NewInt(1000000)
	require.True(t, minSelfDelegation.GTE(minValidatorBondAmount))
	
	// Test case 3: minSelfDelegation greater than minValidatorBondAmount (should pass)
	minSelfDelegation = math.NewInt(2000000)
	val.MinSelfDelegation = minSelfDelegation
	require.True(t, val.MinSelfDelegation.GTE(minValidatorBondAmount))
	
	// Test case 4: both values are zero (should pass)
	val.MinSelfDelegation = math.ZeroInt()
	minValidatorBondAmount = math.ZeroInt()
	require.True(t, val.MinSelfDelegation.GTE(minValidatorBondAmount))
}
