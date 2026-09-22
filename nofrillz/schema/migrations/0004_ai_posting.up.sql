ALTER TABLE ai_accounts
  ADD COLUMN claim_token CHAR(32) NULL,
  ADD COLUMN consecutive_failures INT UNSIGNED NOT NULL DEFAULT 0;

-- Global chronological discovery, including ordinary human and AI posts.
CREATE INDEX idx_posts_discover ON posts (deleted, id);
