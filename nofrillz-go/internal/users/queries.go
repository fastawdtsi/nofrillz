package users

const (
	selectUserByIdQuery = `
	select id,email,username,first_name,last_name,about,account_type,password_hash,password_salt,created,updated,deleted,blocked from users where id=?
	`

	selectUserByIdForRequesterQuery = `
	select
		u.id,
		u.email,
		u.username,
		u.first_name,
		u.last_name,
		u.about,
		u.account_type,
		u.password_hash,
		u.password_salt,
		u.created,
		u.updated,
		u.deleted,
		u.blocked,
		exists(
			select 1 from follows
			where follower_id=u.id and following_id=?
		) as is_follower,
		exists(
			select 1 from follows
			where follower_id=? and following_id=u.id
		) as is_following,
		(
			select count(*)
			from follows
			where following_id=u.id
		) as follower_count,
		(
			select count(*)
			from follows
			where follower_id=u.id
		) as following_count,
		(
			select count(*)
			from posts
			where user_id=u.id and deleted is null
		) as post_count
	from users u
	where u.id=?
	  and u.deleted is null
	  and u.blocked is null;
	`

	selectUserByEmailQuery = `
	select id,email,username,first_name,last_name,about,account_type,password_hash,password_salt,created,updated,deleted,blocked from users where email=?
	`

	selectUsersByUsernamePrefixQuery = `
	select id,email,username,first_name,last_name,about,account_type,password_hash,password_salt,created,updated,deleted,blocked
	from users
	where deleted is null and blocked is null and (
		username like ?
		or first_name like ?
		or last_name like ?
	)
	order by username asc
	limit ?
	`

	selectUsersByUsernamePrefixForRequesterQuery = `
	select
		u.id,
		u.email,
		u.username,
		u.first_name,
		u.last_name,
		u.about,
		u.account_type,
		u.password_hash,
		u.password_salt,
		u.created,
		u.updated,
		u.deleted,
		u.blocked,
		exists(
			select 1 from follows
			where follower_id=u.id and following_id=?
		) as is_follower,
		exists(
			select 1 from follows
			where follower_id=? and following_id=u.id
		) as is_following,
		(
			select count(*)
			from follows
			where following_id=u.id
		) as follower_count,
		(
			select count(*)
			from follows
			where follower_id=u.id
		) as following_count,
		(
			select count(*)
			from posts
			where user_id=u.id and deleted is null
		) as post_count
	from users u
	where u.deleted is null and u.blocked is null and (
		u.username like ?
		or u.first_name like ?
		or u.last_name like ?
	)
	order by u.username asc
	limit ?;
	`

	selectFollowersByUserForRequesterPageQuery = `
	select
		u.id,
		u.email,
		u.username,
		u.first_name,
		u.last_name,
		u.about,
		u.account_type,
		u.password_hash,
		u.password_salt,
		u.created,
		u.updated,
		u.deleted,
		u.blocked,
		exists(
			select 1 from follows
			where follower_id=u.id and following_id=?
		) as is_follower,
		exists(
			select 1 from follows
			where follower_id=? and following_id=u.id
		) as is_following,
		(
			select count(*)
			from follows
			where following_id=u.id
		) as follower_count,
		(
			select count(*)
			from follows
			where follower_id=u.id
		) as following_count,
		(
			select count(*)
			from posts
			where user_id=u.id and deleted is null
		) as post_count
	from follows f
	join users u on u.id=f.follower_id
	where f.following_id=?
	  and u.deleted is null
	  and u.blocked is null
	order by f.follower_id desc
	limit ?;
	`

	selectFollowersByUserForRequesterPageByCursorQuery = `
	select
		u.id,
		u.email,
		u.username,
		u.first_name,
		u.last_name,
		u.about,
		u.account_type,
		u.password_hash,
		u.password_salt,
		u.created,
		u.updated,
		u.deleted,
		u.blocked,
		exists(
			select 1 from follows
			where follower_id=u.id and following_id=?
		) as is_follower,
		exists(
			select 1 from follows
			where follower_id=? and following_id=u.id
		) as is_following,
		(
			select count(*)
			from follows
			where following_id=u.id
		) as follower_count,
		(
			select count(*)
			from follows
			where follower_id=u.id
		) as following_count,
		(
			select count(*)
			from posts
			where user_id=u.id and deleted is null
		) as post_count
	from follows f
	join users u on u.id=f.follower_id
	where f.following_id=?
	  and f.follower_id < ?
	  and u.deleted is null
	  and u.blocked is null
	order by f.follower_id desc
	limit ?;
	`
)
