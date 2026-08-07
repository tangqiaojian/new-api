package model

// IsChannelSubscriptionPoolAvailable reports whether a channel can accept new
// usage under its active shared subscription pool. Channels without an active
// pool are unrestricted.
func IsChannelSubscriptionPoolAvailable(channelId int) (bool, error) {
	pool, err := GetEligibleChannelSubscriptionPool(channelId)
	if err != nil {
		return false, err
	}
	if pool == nil {
		return true, nil
	}
	if pool.AmountTotal > 0 && pool.AmountUsed >= pool.AmountTotal {
		return false, nil
	}
	if pool.TokensTotal > 0 && pool.TokensUsed >= pool.TokensTotal {
		return false, nil
	}
	return true, nil
}

func filterChannelIdsBySubscriptionPool(channels []int) ([]int, error) {
	if len(channels) == 0 {
		return channels, nil
	}
	pools, err := GetEligibleChannelSubscriptionPools(channels)
	if err != nil {
		return nil, err
	}
	filtered := make([]int, 0, len(channels))
	for _, channelId := range channels {
		pool := pools[channelId]
		if pool != nil && ((pool.AmountTotal > 0 && pool.AmountUsed >= pool.AmountTotal) ||
			(pool.TokensTotal > 0 && pool.TokensUsed >= pool.TokensTotal)) {
			continue
		}
		filtered = append(filtered, channelId)
	}
	return filtered, nil
}
