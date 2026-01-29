package v6

import (
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// MigrateStore performs in-place store migrations from v5 to v6.
// The migration ensures that the min_validator_bond_amount field is properly
// initialized in the params. This is necessary because the field was added
// after the initial deployment, and existing chains may have params without
// this field.
func MigrateStore(ctx sdk.Context, store storetypes.KVStore, cdc codec.BinaryCodec) error {
	return migrateParams(store, cdc)
}

// migrateParams migrates the staking params to ensure all fields are present
func migrateParams(store storetypes.KVStore, cdc codec.BinaryCodec) error {
	var oldParams types.Params
	
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		// No params stored, nothing to migrate
		return nil
	}

	if err := cdc.Unmarshal(bz, &oldParams); err != nil {
		return err
	}

	// If MinValidatorBondAmount is already set (non-nil), migration already done
	if !oldParams.MinValidatorBondAmount.IsNil() {
		return nil
	}

	// Create new params with the MinValidatorBondAmount field properly initialized
	newParams := types.NewParams(
		oldParams.UnbondingTime,
		oldParams.MaxValidators,
		oldParams.MaxEntries,
		oldParams.HistoricalEntries,
		oldParams.BondDenom,
		oldParams.MinCommissionRate,
		math.ZeroInt(), // Initialize with zero, can be updated via governance
	)

	// Marshal and store the updated params
	newBz, err := cdc.Marshal(&newParams)
	if err != nil {
		return err
	}

	store.Set(types.ParamsKey, newBz)
	return nil
}
