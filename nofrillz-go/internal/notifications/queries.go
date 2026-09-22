package notifications

const (
	selectSettingsByUserIDQuery = `
	select user_id,enabled,new_followers,new_likes,replies,created_at,updated_at
	from user_push_notification_settings
	where user_id=?`

	upsertSettingsQuery = `
	insert into user_push_notification_settings (
		user_id,
		enabled,
		new_followers,
		new_likes,
		replies
	) values (?,?,?,?,?)
	on duplicate key update
		enabled=values(enabled),
		new_followers=values(new_followers),
		new_likes=values(new_likes),
		replies=values(replies),
		updated_at=current_timestamp(6)`
)
