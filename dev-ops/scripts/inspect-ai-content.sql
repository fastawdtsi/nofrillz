-- Read-only inspection of the local content-account product model.
SELECT u.username,a.id AS account_id,a.content_mode,a.check_interval_seconds,
 a.model_options,a.default_model_option,a.last_checked_at,a.last_check_outcome,
 a.last_generated_at,a.next_generate_at
FROM ai_accounts a JOIN users u ON u.id=a.user_id WHERE u.deleted IS NULL;

SELECT u.username,a.id AS account_id,ci.id AS content_item_id,ci.title,
 v.option_id,v.provider,v.model,v.post_id,v.status,
 CASE WHEN v.post_id IS NULL THEN 'no post'
      WHEN p.deleted IS NULL THEN 'visible' ELSE 'removed' END AS post_state
FROM ai_accounts a JOIN users u ON u.id=a.user_id AND u.deleted IS NULL
JOIN ai_content_items ci ON ci.ai_account_id=a.id
JOIN ai_content_variants v ON v.content_item_id=ci.id
LEFT JOIN posts p ON p.id=v.post_id
ORDER BY ci.id DESC,v.option_id LIMIT 30;

SELECT u.username,u.ai_model_preference,author.username AS following,f.ai_model_override
FROM users u JOIN follows f ON f.follower_id=u.id
JOIN users author ON author.id=f.following_id
WHERE u.username IN ('content_reader_a','content_reader_b')
ORDER BY u.username,author.username;

SELECT ci.id,ci.title,ci.sources FROM ai_content_items ci
JOIN ai_accounts a ON a.id=ci.ai_account_id
WHERE a.content_mode='research' ORDER BY ci.id DESC LIMIT 5;
