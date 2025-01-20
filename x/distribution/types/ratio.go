package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"sigs.k8s.io/yaml"
)

// zero fee pool
func InitialRatio() Ratio {
	return Ratio{
		StakingRewards: sdk.NewDecWithPrec(333333333333333334, 18), // 1/3
		Base:           sdk.NewDecWithPrec(333333333333333333, 18), // 1/3
		Burn:           sdk.NewDecWithPrec(333333333333333333, 18), // 1/3
	}
}

func (p Ratio) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// ValidateGenesis validates the ratio for a genesis state
func (r Ratio) ValidateGenesis() error {
	return r.ValidateRatio()
}

func (r Ratio) ValidateRatio() error {
	if r.StakingRewards.IsNegative() {
		return fmt.Errorf("negative staking rewards in ratio, is %v", r.StakingRewards)
	}
	if r.Base.IsNegative() {
		return fmt.Errorf("negative base in ratio, is %v", r.Base)
	}
	if r.Burn.IsNegative() {
		return fmt.Errorf("negative burn in ratio, is %v", r.Burn)
	}
	sum := r.StakingRewards.Add(r.Base).Add(r.Burn)
	if !sum.Equal(sdk.NewDec(1)) {
		return fmt.Errorf("the ratio should sum up to be 1.0, is %v", sum)
	}

	return nil
}
