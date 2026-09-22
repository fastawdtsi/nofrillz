package bookmarks

const (
	insertBookmarkQuery = `
	insert ignore into post_bookmarks (user_id,post_id) values (?,?)
	`

	deleteBookmarkQuery = `
	delete from post_bookmarks where user_id=? and post_id=?
	`

	selectBookmarkExistsQuery = `
	select exists(
	  select 1
	  from post_bookmarks
	  where user_id=?
	    and post_id=?
	)
	`

	selectBookmarkedPostsQuery = `
	select
		p.id,
		p.user_id,
		p.body,
		p.source,
		p.created,
		p.updated,
		p.deleted,
		u.username,
		u.first_name,
		u.last_name,
		u.account_type,
		l.user_id is not null as liked,
		true as is_bookmarked,
		pb.created as bookmarked_at
	from post_bookmarks pb
	join posts p
	  on p.id = pb.post_id
	 and p.deleted is null
	join users u
	  on u.id = p.user_id
	 and u.deleted is null
	 and u.blocked is null
	left join likes l
	  on l.post_id = p.id
	 and l.user_id = ?
	where pb.user_id = ?
	order by pb.created desc, pb.post_id desc
	limit ?
	`

	selectBookmarkedPostsByCursorQuery = `
	select
		p.id,
		p.user_id,
		p.body,
		p.source,
		p.created,
		p.updated,
		p.deleted,
		u.username,
		u.first_name,
		u.last_name,
		u.account_type,
		l.user_id is not null as liked,
		true as is_bookmarked,
		pb.created as bookmarked_at
	from post_bookmarks pb
	join posts p
	  on p.id = pb.post_id
	 and p.deleted is null
	join users u
	  on u.id = p.user_id
	 and u.deleted is null
	 and u.blocked is null
	left join likes l
	  on l.post_id = p.id
	 and l.user_id = ?
	where pb.user_id = ?
	  and (
		pb.created < ?
		or (pb.created = ? and pb.post_id < ?)
	  )
	order by pb.created desc, pb.post_id desc
	limit ?
	`
)
