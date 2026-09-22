-- Local development cleanup only; never part of automatic migrations.
-- Retire exactly the three paused fictional demo accounts from the superseded
-- implementation. Preserve users/posts/audits for recovery and all other data.
START TRANSACTION;
CREATE TEMPORARY TABLE legacy_demo_ids (id BIGINT UNSIGNED PRIMARY KEY);
INSERT INTO legacy_demo_ids
 SELECT u.id FROM users u JOIN ai_accounts a ON a.user_id=u.id
 WHERE u.account_type='ai' AND a.enabled=FALSE
 AND ((u.username='mara_windowsill' AND u.email='mara.local@example.invalid')
   OR (u.username='dex_deadformat' AND u.email='dex.local@example.invalid')
   OR (u.username='ines_afterhours' AND u.email='ines.local@example.invalid'));
UPDATE ai_accounts a JOIN legacy_demo_ids d ON d.id=a.user_id
 SET a.enabled=FALSE,a.next_generate_at=NULL,a.claim_token=NULL,
     a.generation_status='idle',a.generation_started_at=NULL;
UPDATE posts p JOIN legacy_demo_ids d ON d.id=p.user_id
 SET p.deleted=COALESCE(p.deleted,UTC_TIMESTAMP());
UPDATE users u JOIN legacy_demo_ids d ON d.id=u.id
 SET u.deleted=COALESCE(u.deleted,UTC_TIMESTAMP());
SELECT COUNT(*) AS retired_legacy_demo_accounts FROM legacy_demo_ids;
COMMIT;
DROP TEMPORARY TABLE legacy_demo_ids;
