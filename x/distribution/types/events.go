package types

// distribution module event types
const (
	EventTypeSetWithdrawAddress = "set_withdraw_address"
	EventTypeRewards            = "rewards"
	EventTypeCommission         = "commission"
	EventTypeWithdrawRewards    = "withdraw_rewards"
	EventTypeWithdrawCommission = "withdraw_commission"
	EventTypeProposerReward     = "proposer_reward"
	EventTypeChangeRatio        = "change_ratio"
	EventTypeChangeBaseAddress  = "change_base_address"
	EventTypeChangeModerator    = "change_moderator"
	EventTypeBurnFee            = "burn_fee"
	EventTypeBaseFee            = "base_fee"
	EventTypeStakingRewards     = "staking_rewards"
	EventTypeStakingFee         = "staking_fee"

	AttributeKeyWithdrawAddress = "withdraw_address"
	AttributeKeyValidator       = "validator"
	AttributeKeyDelegator       = "delegator"
	AttributeValueCategory      = ModuleName
)
