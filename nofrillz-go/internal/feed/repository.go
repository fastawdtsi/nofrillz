package feed

import (
	"context"
	"database/sql"
	"nofrillz/internal/aiselection"
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }
func (r *Repository) List(ctx context.Context, userID uint64, cursor *uint64, limit int) ([]*Post, error) {
	return r.list(ctx, userID, cursor, limit, false)
}
func (r *Repository) ListDiscover(ctx context.Context, userID uint64, cursor *uint64, limit int) ([]*Post, error) {
	return r.list(ctx, userID, cursor, limit, true)
}
func (r *Repository) list(ctx context.Context, userID uint64, cursor *uint64, limit int, discover bool) ([]*Post, error) {
	query := `SELECT p.id,p.body,p.source,p.created,l.user_id IS NOT NULL,b.user_id IS NOT NULL,p.user_id,u.username,u.first_name,u.last_name,u.account_type,` + aiselection.SortID + `,ci.id,COALESCE(av.option_id,''),COALESCE(av.provider,''),COALESCE(av.model,'')
 FROM posts p JOIN users u ON u.id=p.user_id AND u.deleted IS NULL AND u.blocked IS NULL
 LEFT JOIN likes l ON l.post_id=p.id AND l.user_id=?
 LEFT JOIN post_bookmarks b ON b.post_id=p.id AND b.user_id=?` + aiselection.Joins + `
 WHERE p.deleted IS NULL` + aiselection.Visible
	args := []any{userID, userID, userID}
	if !discover {
		query += ` AND (p.user_id=? OR af.follower_id IS NOT NULL)`
		args = append(args, userID)
	}
	if cursor != nil {
		query += ` AND ` + aiselection.SortID + ` < ?`
		args = append(args, *cursor)
	}
	query += ` ORDER BY ` + aiselection.SortID + ` DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*Post{}
	for rows.Next() {
		p := &Post{}
		if err = rows.Scan(&p.ID, &p.Body, &p.Source, &p.Created, &p.Liked, &p.IsBookmarked, &p.UserID, &p.Username, &p.FirstName, &p.LastName, &p.AccountType, &p.SortID, &p.ContentItemID, &p.ModelOption, &p.Provider, &p.Model); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
