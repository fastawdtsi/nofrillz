package likes

import "time"

type Like struct {
	PostID  uint64
	UserID  uint64
	Created time.Time
}
