package consensus

func BlockReward(height int64) int64 {
	return CalcBlockSubsidy(height)
}

func GetSubsidyDepth(height int64) int {
	return int(height / int64(RewardInterval))
}

func CalcBlockSubsidy(height int64) int64 {
	halving := GetSubsidyDepth(height)
	subsidy := int64(InitialReward)
	for halving > 0 {
		subsidy /= 2
		halving--
	}
	if subsidy < 0 {
		subsidy = 0
	}
	return subsidy
}
