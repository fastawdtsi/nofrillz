package aiaccounts

const (
	selectAIAccountByIDQuery = `
	select
		id,
		user_id,
		enabled,
		topic,
		description,
		system_prompt,
		style_prompt,
		min_posts_per_day,
		max_posts_per_day,
		next_generate_at,
		last_generated_at,
		generation_status,
		generation_started_at,
		generation_error,
		claim_token,
		consecutive_failures,
 content_mode, check_interval_seconds, COALESCE(exclusions,''),
 source_urls, model_options, default_model_option, source_max_age_hours,
 last_checked_at, COALESCE(last_check_outcome,''),
		created_at,
		updated_at
	from ai_accounts
	where id=?
	`

	selectDueAccountsForClaimQuery = `
	select
		id,
		user_id,
		enabled,
		topic,
		description,
		system_prompt,
		style_prompt,
		min_posts_per_day,
		max_posts_per_day,
		next_generate_at,
		last_generated_at,
		generation_status,
		generation_started_at,
		generation_error,
		claim_token,
		consecutive_failures,
 content_mode, check_interval_seconds, COALESCE(exclusions,''),
 source_urls, model_options, default_model_option, source_max_age_hours,
 last_checked_at, COALESCE(last_check_outcome,''),
		created_at,
		updated_at
	from ai_accounts
	where enabled = true
	  and exists (select 1 from users u where u.id=ai_accounts.user_id and u.blocked is null and u.deleted is null)
	  and next_generate_at is not null
	  and next_generate_at <= ?
	  and (
		generation_status = 'idle'
		or (
			generation_status = 'running'
			and (
				generation_started_at is null
				or generation_started_at <= ?
			)
		)
	  )
	order by next_generate_at asc, id asc
	limit ?
	for update skip locked
	`

	insertAIAccountQuery = `
	insert into ai_accounts (
		id,
		user_id,
		enabled,
		topic,
		description,
		system_prompt,
		style_prompt,
		min_posts_per_day,
		max_posts_per_day,
		next_generate_at,
		last_generated_at,
		generation_status,
		generation_started_at,
		generation_error
	) values (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`

	insertPostGenerationQuery = `
	insert into ai_post_generations (
		id,
		ai_account_id,
		post_id,
		status,
		prompt,
		candidate_body,
		final_body,
		reject_reason,
		error,
		model
	) values (?,?,?,?,?,?,?,?,?,?)
	`

	selectPostGenerationsByAccountIDQuery = `
	select
		id,
		ai_account_id,
		post_id,
		status,
		prompt,
		candidate_body,
		final_body,
		reject_reason,
		error,
		model,
		created_at
	from ai_post_generations
	where ai_account_id=?
	order by created_at desc, id desc
	limit ?
	`

	updateAccountSuccessQuery = `
	update ai_accounts
	set next_generate_at=?,
	    last_generated_at=?,
	    generation_status='idle',
	    generation_started_at=null,
	    generation_error=null,
	    consecutive_failures=0
	where id=? and generation_status <> 'running'
	`

	updateAccountFailureQuery = `
	update ai_accounts
	set next_generate_at=?,
	    generation_status='idle',
	    generation_started_at=null,
	    generation_error=?
	where id=?
	`
)
