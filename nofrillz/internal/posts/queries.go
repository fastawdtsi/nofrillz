package posts

import "nofrillz/internal/aiselection"

const (
	selectPostByIdQuery = `
	select p.id,p.user_id,p.body,p.source,p.created,p.updated,p.deleted,u.username,u.first_name,u.last_name,u.account_type,l.user_id is not null as liked,b.user_id is not null as is_bookmarked
	from posts p
	join users u on u.id=p.user_id
	  and u.deleted is null
	  and u.blocked is null
	left join likes l
	  on l.post_id=p.id
	 and l.user_id=?
	left join post_bookmarks b
	  on b.post_id=p.id
	 and b.user_id=?
	where p.id=?
	  and p.deleted is null
	`

	selectPostsByUserIdQuery = `
	select p.id,p.user_id,p.body,p.source,p.created,p.updated,p.deleted,u.username,u.first_name,u.last_name,u.account_type,l.user_id is not null as liked,b.user_id is not null as is_bookmarked
	from posts p
	join users u on u.id=p.user_id
	  and u.deleted is null
	  and u.blocked is null
	left join likes l
	  on l.post_id=p.id
	 and l.user_id=?
	left join post_bookmarks b
	  on b.post_id=p.id
	 and b.user_id=?
` + aiselection.Joins + `
	where p.user_id=?
	  and p.deleted is null` + aiselection.Visible + `
	order by ` + aiselection.SortID + ` desc
	limit ?
	`

	selectPostsByUserIdBeforeIdQuery = `
	select p.id,p.user_id,p.body,p.source,p.created,p.updated,p.deleted,u.username,u.first_name,u.last_name,u.account_type,l.user_id is not null as liked,b.user_id is not null as is_bookmarked
	from posts p
	join users u on u.id=p.user_id
	  and u.deleted is null
	  and u.blocked is null
	left join likes l
	  on l.post_id=p.id
	 and l.user_id=?
	left join post_bookmarks b
	  on b.post_id=p.id
	 and b.user_id=?
` + aiselection.Joins + `
	where p.user_id=?
	  and p.deleted is null` + aiselection.Visible + `
	  and ` + aiselection.SortID + `<COALESCE((SELECT content_item_id FROM ai_content_variants WHERE post_id=?),?)
	order by ` + aiselection.SortID + ` desc
	limit ?
	`
)
