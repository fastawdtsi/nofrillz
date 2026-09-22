package sessions

const (
	insertSessionQuery = `
	insert into sessions (id,user_id) values (?,?)
	`

	selectActiveSessionByIDQuery = `
	select id,user_id,created,updated,revoked
	from sessions
	where id=? and revoked is null
	limit 1
	`

	revokeSessionByIDQuery = `
	update sessions set revoked=utc_timestamp(6) where id=? and revoked is null
	`

	insertRefreshSessionQuery = `
	insert into session_refresh_tokens (token_hash,session_id,expires_at) values (?,?,?)
	`

	selectActiveRefreshSessionByTokenQuery = `
	select r.token_hash,r.session_id,s.user_id,r.created,r.expires_at,r.revoked
	from session_refresh_tokens r
	join sessions s on s.id = r.session_id
	where r.token_hash=? and r.revoked is null and r.expires_at > utc_timestamp(6) and s.revoked is null
	limit 1
	`

	revokeRefreshSessionByTokenQuery = `
	update session_refresh_tokens set revoked=utc_timestamp(6) where token_hash=? and revoked is null
	`

	revokeRefreshSessionsBySessionIDQuery = `
	update session_refresh_tokens
	set revoked=utc_timestamp(6)
	where session_id=? and revoked is null
	`
)
