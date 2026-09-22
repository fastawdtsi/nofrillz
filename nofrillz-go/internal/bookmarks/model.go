package bookmarks

import "time"

type Cursor struct {
	Created time.Time
	PostID  uint64
}
