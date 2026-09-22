package follows

const (
	insertFollowQuery = `
	insert ignore into follows (follower_id,following_id) values (?,?)
	`

	deleteFollowQuery = `
	delete from follows where follower_id=? and following_id=?
	`

	selectFollowersQuery = `
	select follower_id from follows where following_id=? order by created desc
	`

	selectFollowersPageQuery = `
	select follower_id
	from follows
	where following_id=?
	order by follower_id desc
	limit ?
	`

	selectFollowersPageByCursorQuery = `
	select follower_id
	from follows
	where following_id=?
	  and follower_id < ?
	order by follower_id desc
	limit ?
	`

	selectFollowingQuery = `
	select following_id from follows where follower_id=? order by created desc
	`
)
