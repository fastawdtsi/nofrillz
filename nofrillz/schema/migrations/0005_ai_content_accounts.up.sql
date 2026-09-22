-- Content missions reuse topic, description (mission), system_prompt (additional
-- instructions), style_prompt, and the normal AI user's public name/bio.
ALTER TABLE ai_accounts
 ADD COLUMN content_mode VARCHAR(16) NOT NULL DEFAULT 'generative',
 ADD COLUMN check_interval_seconds INT UNSIGNED NOT NULL DEFAULT 86400,
 ADD COLUMN exclusions TEXT NULL,
 ADD COLUMN source_urls JSON NULL,
 ADD COLUMN model_options JSON NULL,
 ADD COLUMN default_model_option VARCHAR(64) NOT NULL DEFAULT 'openai',
 ADD COLUMN source_max_age_hours INT UNSIGNED NOT NULL DEFAULT 168,
 ADD COLUMN last_checked_at TIMESTAMP(6) NULL,
 ADD COLUMN last_check_outcome VARCHAR(64) NULL;
UPDATE ai_accounts SET model_options=JSON_ARRAY('openai'), source_urls=JSON_ARRAY(),
 check_interval_seconds=GREATEST(300, FLOOR(86400 / GREATEST(1,(min_posts_per_day+max_posts_per_day)/2)));
ALTER TABLE users ADD COLUMN ai_model_preference VARCHAR(64) NULL;
ALTER TABLE follows ADD COLUMN ai_model_override VARCHAR(64) NULL;

CREATE TABLE ai_content_items (
 id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
 ai_account_id BIGINT UNSIGNED NOT NULL,
 dedup_key CHAR(64) NOT NULL,
 title VARCHAR(512) NOT NULL,
 context TEXT NOT NULL,
 sources JSON NOT NULL,
 status VARCHAR(16) NOT NULL DEFAULT 'processing',
 created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 published_at TIMESTAMP(6) NULL,
 completed_at TIMESTAMP(6) NULL,
 UNIQUE KEY uq_ai_content_source (ai_account_id,dedup_key),
 KEY idx_ai_content_pending (ai_account_id,status,id),
 CONSTRAINT fk_ai_content_account FOREIGN KEY (ai_account_id) REFERENCES ai_accounts(id)
);
CREATE TABLE ai_content_variants (
 content_item_id BIGINT UNSIGNED NOT NULL,
 option_id VARCHAR(64) NOT NULL,
 provider VARCHAR(64) NOT NULL,
 model VARCHAR(128) NOT NULL,
 post_id BIGINT UNSIGNED NULL,
 status VARCHAR(16) NOT NULL,
 error VARCHAR(512) NULL,
 created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 PRIMARY KEY (content_item_id,option_id),
 UNIQUE KEY uq_ai_variant_post (post_id),
 CONSTRAINT fk_ai_variant_content FOREIGN KEY(content_item_id) REFERENCES ai_content_items(id),
 CONSTRAINT fk_ai_variant_post FOREIGN KEY(post_id) REFERENCES posts(id)
);
CREATE TABLE ai_processed_sources (
 ai_account_id BIGINT UNSIGNED NOT NULL,
 fingerprint CHAR(64) NOT NULL,
 content_item_id BIGINT UNSIGNED NULL,
 outcome VARCHAR(32) NOT NULL,
 checked_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
 PRIMARY KEY(ai_account_id,fingerprint),
 CONSTRAINT fk_ai_processed_account FOREIGN KEY(ai_account_id) REFERENCES ai_accounts(id),
 CONSTRAINT fk_ai_processed_item FOREIGN KEY(content_item_id) REFERENCES ai_content_items(id)
);
