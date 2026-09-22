package likes

const (
	insertLikeQuery = `
	insert ignore into likes (post_id,user_id) values (?,?)
	`

	deleteLikeQuery = `
	delete from likes where post_id=? and user_id=?
	`
)
