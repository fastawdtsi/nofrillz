package feed

import "context"

type repository interface {
	List(ctx context.Context, requesterUserID uint64, cursor *uint64, limit int) ([]*Post, error)
	ListDiscover(ctx context.Context, requesterUserID uint64, cursor *uint64, limit int) ([]*Post, error)
}

type Service struct {
	repository repository
}

func NewService(repository repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) List(ctx context.Context, requesterUserID uint64, cursor *uint64, limit int) ([]*Post, *uint64, error) {
	posts, err := s.repository.List(ctx, requesterUserID, cursor, limit+1)
	if err != nil {
		return nil, nil, err
	}

	if len(posts) <= limit {
		return posts, nil, nil
	}

	posts = posts[:limit]
	lastPost := posts[len(posts)-1]
	nextCursor := lastPost.SortID
	if nextCursor == 0 {
		nextCursor = lastPost.ID
	}

	return posts, &nextCursor, nil
}

func (s *Service) Discover(ctx context.Context, requesterUserID uint64, cursor *uint64, limit int) ([]*Post, *uint64, error) {
	items, err := s.repository.ListDiscover(ctx, requesterUserID, cursor, limit+1)
	if err != nil {
		return nil, nil, err
	}
	if len(items) <= limit {
		return items, nil, nil
	}
	items = items[:limit]
	next := items[len(items)-1].SortID
	if next == 0 {
		next = items[len(items)-1].ID
	}
	return items, &next, nil
}
