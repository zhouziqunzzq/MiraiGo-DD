package bili

// These are APIs exposed to other modules.
// They should only be called after initialization of all modules.

func GetSubscriptionByGroupId(gid int64) []UserInfo {
	if instance == nil {
		return nil
	}

	// get bid list by group id
	instance.subscriptionRwMu.RLock()
	bidListCopy := make([]int64, 0)
	if bidList, ok := instance.groupIdToBiliUidList[gid]; ok {
		bidListCopy = append(bidListCopy, bidList...)
	}
	instance.subscriptionRwMu.RUnlock()

	if len(bidListCopy) == 0 {
		return nil
	}

	// get user info list by bid list
	instance.infoBufRwMu.RLock()
	defer instance.infoBufRwMu.RUnlock()
	infoList := make([]UserInfo, 0, len(bidListCopy))
	for _, bid := range bidListCopy {
		if info, ok := instance.biliUserInfoBuf[bid]; ok {
			infoList = append(infoList, *info)
		} else {
			// no info in buf for now, just use its bid
			infoList = append(infoList, UserInfo{Mid: int(bid)})
		}
	}
	return infoList
}
