// Package aiselection shares the exact same model resolution across feed and
// profile history. Likes/bookmarks and direct post links retain variant identity.
package aiselection

const Joins = `
 LEFT JOIN ai_content_variants av ON av.post_id=p.id
 LEFT JOIN ai_content_items ci ON ci.id=av.content_item_id
 LEFT JOIN ai_accounts aa ON aa.id=ci.ai_account_id
 LEFT JOIN users viewer ON viewer.id=?
 LEFT JOIN follows af ON af.follower_id=viewer.id AND af.following_id=p.user_id
`
const Visible = ` AND (av.post_id IS NULL OR (
 ci.published_at IS NOT NULL AND p.id=(
 SELECT candidate.post_id FROM ai_content_variants candidate
 JOIN posts cp ON cp.id=candidate.post_id AND cp.deleted IS NULL
 WHERE candidate.content_item_id=ci.id AND candidate.status='published'
 ORDER BY CASE
 WHEN candidate.option_id=af.ai_model_override THEN 0
 WHEN candidate.option_id=viewer.ai_model_preference THEN 1
 WHEN candidate.option_id=aa.default_model_option THEN 2
 ELSE 3 END, candidate.option_id ASC
 LIMIT 1)))`
const SortID = `COALESCE(ci.id,p.id)`
