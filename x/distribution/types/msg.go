package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// distribution message types
const (
	TypeMsgSetWithdrawAddress          = "set_withdraw_address"
	TypeMsgWithdrawDelegatorReward     = "withdraw_delegator_reward"
	TypeMsgWithdrawValidatorCommission = "withdraw_validator_commission"
	TypeMsgFundCommunityPool           = "fund_community_pool"
	TypeMsgChangeRatio                 = "change_ratio"
	TypeMsgChangeBaseAddress           = "change_base_address"
	TypeMsgChangeModerator             = "change_moderator"
)

// Verify interface at compile time
var _, _, _ sdk.Msg = &MsgSetWithdrawAddress{}, &MsgWithdrawDelegatorReward{}, &MsgWithdrawValidatorCommission{}

func NewMsgSetWithdrawAddress(delAddr, withdrawAddr sdk.AccAddress) *MsgSetWithdrawAddress {
	return &MsgSetWithdrawAddress{
		DelegatorAddress: delAddr.String(),
		WithdrawAddress:  withdrawAddr.String(),
	}
}

func (msg MsgSetWithdrawAddress) Route() string { return ModuleName }
func (msg MsgSetWithdrawAddress) Type() string  { return TypeMsgSetWithdrawAddress }

// Return address that must sign over msg.GetSignBytes()
func (msg MsgSetWithdrawAddress) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(msg.DelegatorAddress)
	return []sdk.AccAddress{delegator}
}

// get the bytes for the message signer to sign on
func (msg MsgSetWithdrawAddress) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// quick validity check
func (msg MsgSetWithdrawAddress) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.DelegatorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid delegator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.WithdrawAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid withdraw address: %s", err)
	}

	return nil
}

func NewMsgWithdrawDelegatorReward(delAddr sdk.AccAddress, valAddr sdk.ValAddress) *MsgWithdrawDelegatorReward {
	return &MsgWithdrawDelegatorReward{
		DelegatorAddress: delAddr.String(),
		ValidatorAddress: valAddr.String(),
	}
}

func (msg MsgWithdrawDelegatorReward) Route() string { return ModuleName }
func (msg MsgWithdrawDelegatorReward) Type() string  { return TypeMsgWithdrawDelegatorReward }

// Return address that must sign over msg.GetSignBytes()
func (msg MsgWithdrawDelegatorReward) GetSigners() []sdk.AccAddress {
	delegator, _ := sdk.AccAddressFromBech32(msg.DelegatorAddress)
	return []sdk.AccAddress{delegator}
}

// get the bytes for the message signer to sign on
func (msg MsgWithdrawDelegatorReward) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// quick validity check
func (msg MsgWithdrawDelegatorReward) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.DelegatorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid delegator address: %s", err)
	}
	if _, err := sdk.ValAddressFromBech32(msg.ValidatorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid validator address: %s", err)
	}
	return nil
}

func NewMsgWithdrawValidatorCommission(valAddr sdk.ValAddress) *MsgWithdrawValidatorCommission {
	return &MsgWithdrawValidatorCommission{
		ValidatorAddress: valAddr.String(),
	}
}

func (msg MsgWithdrawValidatorCommission) Route() string { return ModuleName }
func (msg MsgWithdrawValidatorCommission) Type() string  { return TypeMsgWithdrawValidatorCommission }

// Return address that must sign over msg.GetSignBytes()
func (msg MsgWithdrawValidatorCommission) GetSigners() []sdk.AccAddress {
	valAddr, _ := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	return []sdk.AccAddress{sdk.AccAddress(valAddr)}
}

// get the bytes for the message signer to sign on
func (msg MsgWithdrawValidatorCommission) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// quick validity check
func (msg MsgWithdrawValidatorCommission) ValidateBasic() error {
	if _, err := sdk.ValAddressFromBech32(msg.ValidatorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid validator address: %s", err)
	}
	return nil
}

// NewMsgFundCommunityPool returns a new MsgFundCommunityPool with a sender and
// a funding amount.
func NewMsgFundCommunityPool(amount sdk.Coins, depositor sdk.AccAddress) *MsgFundCommunityPool {
	return &MsgFundCommunityPool{
		Amount:    amount,
		Depositor: depositor.String(),
	}
}

// Route returns the MsgFundCommunityPool message route.
func (msg MsgFundCommunityPool) Route() string { return ModuleName }

// Type returns the MsgFundCommunityPool message type.
func (msg MsgFundCommunityPool) Type() string { return TypeMsgFundCommunityPool }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgFundCommunityPool) GetSigners() []sdk.AccAddress {
	depositor, _ := sdk.AccAddressFromBech32(msg.Depositor)
	return []sdk.AccAddress{depositor}
}

// GetSignBytes returns the raw bytes for a MsgFundCommunityPool message that
// the expected signer needs to sign.
func (msg MsgFundCommunityPool) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgFundCommunityPool message validation.
func (msg MsgFundCommunityPool) ValidateBasic() error {
	if !msg.Amount.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}
	if _, err := sdk.AccAddressFromBech32(msg.Depositor); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid depositor address: %s", err)
	}
	return nil
}

// NewMsgChangeRatio returns a new MsgChangeRatio with a new distribution ratio
func NewMsgChangeRatio(moderator sdk.AccAddress, ratio Ratio) *MsgChangeRatio {
	return &MsgChangeRatio{
		ModeratorAddress: moderator.String(),
		Ratio:            ratio,
	}
}

// Route returns the MsgChangeRatio message route.
func (msg MsgChangeRatio) Route() string { return ModuleName }

// Type returns the MsgChangeRatio message type.
func (msg MsgChangeRatio) Type() string { return TypeMsgChangeRatio }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgChangeRatio) GetSigners() []sdk.AccAddress {
	moderator, _ := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	return []sdk.AccAddress{moderator}
}

// GetSignBytes returns the raw bytes for a MsgChangeRatio message that
// the expected signer needs to sign.
func (msg MsgChangeRatio) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgChangeRatio message validation.
func (msg MsgChangeRatio) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ModeratorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid moderator address: %s", err)
	}
	if err := msg.Ratio.ValidateRatio(); err != nil {
		return ErrInvalidRatio.Wrapf("%s", err)
	}
	return nil
}

// NewMsgChangeBaseAddress returns a new MsgChangeBaseAddress with a new base address
func NewMsgChangeBaseAddress(moderator sdk.AccAddress, newBaseAddress sdk.AccAddress) *MsgChangeBaseAddress {
	return &MsgChangeBaseAddress{
		ModeratorAddress: moderator.String(),
		NewBaseAddress:   newBaseAddress.String(),
	}
}

// Route returns the MsgChangeBaseAddress message route.
func (msg MsgChangeBaseAddress) Route() string { return ModuleName }

// Type returns the MsgChangeBaseAddress message type.
func (msg MsgChangeBaseAddress) Type() string { return TypeMsgChangeBaseAddress }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgChangeBaseAddress) GetSigners() []sdk.AccAddress {
	moderator, _ := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	return []sdk.AccAddress{moderator}
}

// GetSignBytes returns the raw bytes for a MsgChangeBaseAddress message that
// the expected signer needs to sign.
func (msg MsgChangeBaseAddress) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgChangeBaseAddress message validation.
func (msg MsgChangeBaseAddress) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ModeratorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid moderator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewBaseAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid new base address: %s", err)
	}
	return nil
}

// NewMsgChangeModerator returns a new MsgChangeModerator with a new moderator
func NewMsgChangeModerator(moderator sdk.AccAddress, newModerator sdk.AccAddress) *MsgChangeModerator {
	return &MsgChangeModerator{
		ModeratorAddress:    moderator.String(),
		NewModeratorAddress: newModerator.String(),
	}
}

// Route returns the MsgChangeModerator message route.
func (msg MsgChangeModerator) Route() string { return ModuleName }

// Type returns the MsgChangeModerator message type.
func (msg MsgChangeModerator) Type() string { return TypeMsgChangeModerator }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgChangeModerator) GetSigners() []sdk.AccAddress {
	moderator, _ := sdk.AccAddressFromBech32(msg.ModeratorAddress)
	return []sdk.AccAddress{moderator}
}

// GetSignBytes returns the raw bytes for a MsgChangeModerator message that
// the expected signer needs to sign.
func (msg MsgChangeModerator) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgChangeModerator message validation.
func (msg MsgChangeModerator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.ModeratorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid moderator address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewModeratorAddress); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid new moderator address: %s", err)
	}
	return nil
}
