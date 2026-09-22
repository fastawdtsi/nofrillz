package follows

import "time"

type Follow struct {
	FollowerID  uint64 `json:"follower_id"`
	FollowingID uint64 `json:"following_id"`
	Created     time.Time
}

type Follower struct {
	FollowerID uint64
}
