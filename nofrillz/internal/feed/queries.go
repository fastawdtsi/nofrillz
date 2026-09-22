package feed

const (
	selectFeedPostsQuery = `
	SELECT p.id, p.body, p.source, p.created, l.user_id is not null as liked, b.user_id is not null as is_bookmarked, p.user_id, u.username, u.first_name, u.last_name, u.account_type
	FROM posts p
	LEFT JOIN likes l
	ON l.post_id = p.id
	AND l.user_id = ?
	LEFT JOIN post_bookmarks b
	ON b.post_id = p.id
	AND b.user_id = ?
	JOIN users u
	ON u.id = p.user_id
	AND u.deleted IS NULL
	AND u.blocked IS NULL
	WHERE p.deleted IS NULL
	AND (
		p.user_id = ?
		OR EXISTS (
			SELECT 1
			FROM follows f
			WHERE f.follower_id = ?
			AND f.following_id = p.user_id
		)
	)
	ORDER BY p.id DESC
	LIMIT ?;
	`

	selectFeedPostsByCursorQuery = `
	select p.id,p.body,p.source,p.created,l.user_id is not null as liked,b.user_id is not null as is_bookmarked,p.user_id,u.username,u.first_name,u.last_name,u.account_type
	from posts p
	left join likes l
	on l.post_id = p.id
	and l.user_id = ?
	left join post_bookmarks b
	on b.post_id = p.id
	and b.user_id = ?
	join users u
	on u.id = p.user_id
	and u.deleted is null
	and u.blocked is null
	where p.deleted is null
	  and (
		p.user_id = ?
		or exists (
			select 1
			from follows f
			where f.follower_id = ?
			  and f.following_id = p.user_id
		)
	  )
	  and p.id < ?
	order by p.id desc
	limit ?
	`
)

const selectDiscoverPostsQuery = `SELECT p.id,p.body,p.source,p.created,
 l.user_id IS NOT NULL,b.user_id IS NOT NULL,p.user_id,u.username,u.first_name,u.last_name,u.account_type
 FROM posts p
 JOIN users u ON u.id=p.user_id AND u.deleted IS NULL AND u.blocked IS NULL
 LEFT JOIN likes l ON l.post_id=p.id AND l.user_id=?
 LEFT JOIN post_bookmarks b ON b.post_id=p.id AND b.user_id=?
 WHERE p.deleted IS NULL`
