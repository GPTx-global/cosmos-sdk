package v6_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/testutil"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	v6 "github.com/cosmos/cosmos-sdk/x/staking/migrations/v6"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
)

func TestMigrateParams(t *testing.T) {
	cdc := moduletestutil.MakeTestEncodingConfig().Codec
	storeKey := storetypes.NewKVStoreKey("staking")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("transient_test"))
	store := ctx.KVStore(storeKey)

	// Create old params without MinValidatorBondAmount
	oldParams := types.Params{
		UnbondingTime:     time.Hour * 504,
		MaxValidators:     100,
		MaxEntries:        7,
		HistoricalEntries: 10000,
		BondDenom:         "agxn",
		MinCommissionRate: math.LegacyZeroDec(),
		// MinValidatorBondAmount is not set (will be nil after unmarshal)
	}

	// Marshal and store old params
	bz, err := cdc.Marshal(&oldParams)
	require.NoError(t, err)
	store.Set(types.ParamsKey, bz)

	// Run migration
	err = v6.MigrateStore(ctx, store, cdc)
	require.NoError(t, err)

	// Verify migration
	var newParams types.Params
	bz = store.Get(types.ParamsKey)
	require.NotNil(t, bz)
	err = cdc.Unmarshal(bz, &newParams)
	require.NoError(t, err)

	// Check that MinValidatorBondAmount is now set to zero
	require.False(t, newParams.MinValidatorBondAmount.IsNil())
	require.True(t, newParams.MinValidatorBondAmount.Equal(math.ZeroInt()))
	
	// Check other fields remain unchanged
	require.Equal(t, oldParams.UnbondingTime, newParams.UnbondingTime)
	require.Equal(t, oldParams.MaxValidators, newParams.MaxValidators)
	require.Equal(t, oldParams.MaxEntries, newParams.MaxEntries)
	require.Equal(t, oldParams.HistoricalEntries, newParams.HistoricalEntries)
	require.Equal(t, oldParams.BondDenom, newParams.BondDenom)
	require.Equal(t, oldParams.MinCommissionRate, newParams.MinCommissionRate)
}

func TestMigrateParamsAlreadyMigrated(t *testing.T) {
	cdc := moduletestutil.MakeTestEncodingConfig().Codec
	storeKey := storetypes.NewKVStoreKey("staking")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("transient_test"))
	store := ctx.KVStore(storeKey)

	// Create params with MinValidatorBondAmount already set
	alreadyMigratedParams := types.NewParams(
		time.Hour*504,
		100,
		7,
		10000,
		"agxn",
		math.LegacyZeroDec(),
		math.NewInt(1000000),
	)

	// Marshal and store
	bz, err := cdc.Marshal(&alreadyMigratedParams)
	require.NoError(t, err)
	store.Set(types.ParamsKey, bz)

	// Run migration
	err = v6.MigrateStore(ctx, store, cdc)
	require.NoError(t, err)

	// Verify params remain unchanged
	var newParams types.Params
	bz = store.Get(types.ParamsKey)
	require.NotNil(t, bz)
	err = cdc.Unmarshal(bz, &newParams)
	require.NoError(t, err)

	// MinValidatorBondAmount should still be 1000000
	require.True(t, newParams.MinValidatorBondAmount.Equal(math.NewInt(1000000)))
}

func TestMigrateNoParams(t *testing.T) {
	cdc := moduletestutil.MakeTestEncodingConfig().Codec
	storeKey := storetypes.NewKVStoreKey("staking")
	ctx := testutil.DefaultContext(storeKey, storetypes.NewTransientStoreKey("transient_test"))
	store := ctx.KVStore(storeKey)

	// No params stored
	// Run migration
	err := v6.MigrateStore(ctx, store, cdc)
	require.NoError(t, err)

	// Verify no params are stored
	bz := store.Get(types.ParamsKey)
	require.Nil(t, bz)
}
