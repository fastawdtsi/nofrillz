package apns

const (
	selectTokensByUserIDQuery = `
	select token,user_id,created_at,updated_at
	from users_apns_device_tokens
	where user_id=?
	order by created_at asc`

	upsertTokenQuery = `
	insert into users_apns_device_tokens (token,user_id)
	values (?,?)
	on duplicate key update
		user_id=values(user_id),
		updated_at=current_timestamp(6)`
)
