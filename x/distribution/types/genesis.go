package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

//nolint:interfacer
func NewGenesisState(
	params Params, fp FeePool, dwis []DelegatorWithdrawInfo, pp sdk.ConsAddress, r []ValidatorOutstandingRewardsRecord,
	acc []ValidatorAccumulatedCommissionRecord, historical []ValidatorHistoricalRewardsRecord,
	cur []ValidatorCurrentRewardsRecord, dels []DelegatorStartingInfoRecord, slashes []ValidatorSlashEventRecord,
	ratio Ratio, base_addr, moderator string,
) *GenesisState {
	return &GenesisState{
		Params:                          params,
		FeePool:                         fp,
		DelegatorWithdrawInfos:          dwis,
		PreviousProposer:                pp.String(),
		OutstandingRewards:              r,
		ValidatorAccumulatedCommissions: acc,
		ValidatorHistoricalRewards:      historical,
		ValidatorCurrentRewards:         cur,
		DelegatorStartingInfos:          dels,
		ValidatorSlashEvents:            slashes,
		Ratio:                           ratio,
		BaseAddress:                     base_addr,
		ModeratorAddress:                moderator,
	}
}

// get raw genesis raw message for testing
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		FeePool:                         InitialFeePool(),
		Params:                          DefaultParams(),
		DelegatorWithdrawInfos:          []DelegatorWithdrawInfo{},
		PreviousProposer:                "",
		OutstandingRewards:              []ValidatorOutstandingRewardsRecord{},
		ValidatorAccumulatedCommissions: []ValidatorAccumulatedCommissionRecord{},
		ValidatorHistoricalRewards:      []ValidatorHistoricalRewardsRecord{},
		ValidatorCurrentRewards:         []ValidatorCurrentRewardsRecord{},
		DelegatorStartingInfos:          []DelegatorStartingInfoRecord{},
		ValidatorSlashEvents:            []ValidatorSlashEventRecord{},
		Ratio:                           InitialRatio(),
		BaseAddress:                     "",
		ModeratorAddress:                "",
	}
}

// ValidateGenesis validates the genesis state of distribution genesis input
func ValidateGenesis(gs *GenesisState) error {
	if err := validateAddress(gs.ModeratorAddress); err != nil {
		return err
	}
	if err := validateAddress(gs.BaseAddress); err != nil {
		return err
	}
	if err := gs.Params.ValidateBasic(); err != nil {
		return err
	}
	if err := gs.Ratio.ValidateGenesis(); err != nil {
		return err
	}
	return gs.FeePool.ValidateGenesis()
}

// method validates the address for genesis state
func validateAddress(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidType, " %+v", i.(string))
	}

	_, err := sdk.AccAddressFromBech32(v)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, " %+v", err)
	}

	return nil
}
