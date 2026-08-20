package store

import (
	"context"
	"sort"
	"time"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/validate"
)

func (s *MemoryStore) CreatePost(ctx context.Context, p model.HelpPost) (model.HelpPost, error) {
	if err := ctx.Err(); err != nil {
		return model.HelpPost{}, err
	}
	if p.Status == "" {
		p.Status = model.PostDraft
	}
	s.mu.Lock()
	now := s.now()
	p.ID = s.genID(model.PostIDPrefix)
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Status == model.PostOpen {
		t := now
		p.PublishedAt = &t
	}
	s.posts[p.ID] = p
	s.mu.Unlock()
	s.afterWrite()
	return p, nil
}

func (s *MemoryStore) GetPost(ctx context.Context, id string) (model.HelpPost, error) {
	if err := ctx.Err(); err != nil {
		return model.HelpPost{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.posts[id]
	if !ok {
		return model.HelpPost{}, model.ErrNotFound
	}
	return p, nil
}

func (s *MemoryStore) ListPosts(ctx context.Context, f model.PostFilter) ([]model.HelpPost, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.HelpPost, 0)
	for _, p := range s.posts {
		if f.Plaza && !p.Status.VisibleToPlaza() {
			continue
		}
		if f.Type != "" && p.Type != f.Type {
			continue
		}
		if f.Status != "" && p.Status != f.Status {
			continue
		}
		if f.Category != "" && p.Category != f.Category {
			continue
		}
		if f.Urgency != "" && p.Urgency != f.Urgency {
			continue
		}
		if f.Building != "" && !validate.ContainsFold(p.Building, f.Building) {
			continue
		}
		if f.AuthorID != "" && p.AuthorID != f.AuthorID {
			continue
		}
		if f.Query != "" {
			blob := p.Title + " " + p.Content + " " + p.LocationNote
			if !matchQuery(blob, f.Query) {
				continue
			}
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Urgency.Rank() != out[j].Urgency.Rank() {
			return out[i].Urgency.Rank() > out[j].Urgency.Rank()
		}
		ai := s.users[out[i].AuthorID].CreditLevel().Rank()
		aj := s.users[out[j].AuthorID].CreditLevel().Rank()
		if ai != aj {
			return ai > aj
		}
		ti, tj := out[i].PublishedAt, out[j].PublishedAt
		if ti != nil && tj != nil && !ti.Equal(*tj) {
			return ti.After(*tj)
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *MemoryStore) UpdatePost(ctx context.Context, p model.HelpPost) (model.HelpPost, error) {
	if err := ctx.Err(); err != nil {
		return model.HelpPost{}, err
	}
	s.mu.Lock()
	old, ok := s.posts[p.ID]
	if !ok {
		s.mu.Unlock()
		return model.HelpPost{}, model.ErrNotFound
	}
	p.CreatedAt = old.CreatedAt
	p.AuthorID = old.AuthorID
	p.UpdatedAt = s.now()
	s.posts[p.ID] = p
	s.mu.Unlock()
	s.afterWrite()
	return p, nil
}

func (s *MemoryStore) CountActivePostsByAuthor(ctx context.Context, authorID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, p := range s.posts {
		if p.AuthorID == authorID && p.Status.IsActive() {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) CountPostsByStatus(ctx context.Context) (map[model.PostStatus]int, int, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := map[model.PostStatus]int{}
	total := 0
	for _, p := range s.posts {
		m[p.Status]++
		total++
	}
	return m, total, nil
}

func (s *MemoryStore) CountPostsCreatedOn(ctx context.Context, day time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, p := range s.posts {
		if model.ClockDateEqual(p.CreatedAt, day) {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) ExpireOpenPost(ctx context.Context, postID string) (model.HelpPost, error) {
	if err := ctx.Err(); err != nil {
		return model.HelpPost{}, err
	}
	s.mu.Lock()
	p, ok := s.posts[postID]
	if !ok {
		s.mu.Unlock()
		return model.HelpPost{}, model.ErrNotFound
	}
	if p.Status != model.PostOpen {
		s.mu.Unlock()
		return p, nil
	}
	now := s.now()
	p.Status = model.PostExpired
	p.UpdatedAt = now
	p.ClosedReason = "time window ended"
	s.posts[postID] = p
	for _, a := range s.apps {
		if a.PostID == postID && a.Status == model.AppPending {
			a.Status = model.AppExpired
			a.UpdatedAt = now
			s.apps[a.ID] = a
		}
	}
	s.mu.Unlock()
	s.afterWrite()
	return p, nil
}
