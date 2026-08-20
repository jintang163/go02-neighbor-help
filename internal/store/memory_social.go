package store

import (
	"context"
	"sort"
	"time"

	"go02-neighbor-help/internal/model"
)

func (s *MemoryStore) CreateReview(ctx context.Context, r model.Review) (model.Review, error) {
	if err := ctx.Err(); err != nil {
		return model.Review{}, err
	}
	s.mu.Lock()
	if _, exists := s.reviewIdx[reviewKey(r.TaskID, r.FromUserID)]; exists {
		s.mu.Unlock()
		return model.Review{}, model.ErrAlreadyReviewed
	}
	now := s.now()
	r.ID = s.genID(model.ReviewIDPrefix)
	r.CreatedAt = now
	s.reviews[r.ID] = r
	s.reviewIdx[reviewKey(r.TaskID, r.FromUserID)] = r.ID
	if u, ok := s.users[r.ToUserID]; ok {
		u.ReviewCount++
		u.ReviewSum += r.Score
		u.UpdatedAt = now
		s.users[r.ToUserID] = u
	}
	if t, ok := s.tasks[r.TaskID]; ok {
		if r.FromUserID == t.RequesterID {
			t.RequesterReviewed = true
		}
		if r.FromUserID == t.HelperID {
			t.HelperReviewed = true
		}
		t.BothReviewed = t.RequesterReviewed && t.HelperReviewed
		t.UpdatedAt = now
		s.tasks[r.TaskID] = t
	}
	s.mu.Unlock()
	s.afterWrite()
	return r, nil
}

func (s *MemoryStore) GetReview(ctx context.Context, id string) (model.Review, error) {
	if err := ctx.Err(); err != nil {
		return model.Review{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.reviews[id]
	if !ok {
		return model.Review{}, model.ErrNotFound
	}
	return r, nil
}

func (s *MemoryStore) GetReviewByTaskFrom(ctx context.Context, taskID, fromUserID string) (model.Review, error) {
	if err := ctx.Err(); err != nil {
		return model.Review{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.reviewIdx[reviewKey(taskID, fromUserID)]
	if !ok {
		return model.Review{}, model.ErrNotFound
	}
	return s.reviews[id], nil
}

func (s *MemoryStore) ListReviewsByTask(ctx context.Context, taskID string) ([]model.Review, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Review, 0)
	for _, r := range s.reviews {
		if r.TaskID == taskID {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) ListReviewsToUser(ctx context.Context, userID string) ([]model.Review, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Review, 0)
	for _, r := range s.reviews {
		if r.ToUserID == userID && !r.Hidden {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) UpdateReview(ctx context.Context, r model.Review) (model.Review, error) {
	if err := ctx.Err(); err != nil {
		return model.Review{}, err
	}
	s.mu.Lock()
	if _, ok := s.reviews[r.ID]; !ok {
		s.mu.Unlock()
		return model.Review{}, model.ErrNotFound
	}
	s.reviews[r.ID] = r
	s.mu.Unlock()
	s.afterWrite()
	return r, nil
}

func (s *MemoryStore) CountReviews(ctx context.Context) (count int, sum int, err error) {
	if err = ctx.Err(); err != nil {
		return 0, 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.reviews {
		if r.Hidden {
			continue
		}
		count++
		sum += r.Score
	}
	return count, sum, nil
}

func (s *MemoryStore) CreateMessage(ctx context.Context, m model.Message) (model.Message, error) {
	if err := ctx.Err(); err != nil {
		return model.Message{}, err
	}
	s.mu.Lock()
	if _, ok := s.posts[m.PostID]; !ok {
		s.mu.Unlock()
		return model.Message{}, model.ErrNotFound
	}
	if m.ParentID != "" {
		if _, ok := s.messages[m.ParentID]; !ok {
			s.mu.Unlock()
			return model.Message{}, model.ErrNotFound
		}
	}
	m.ID = s.genID(model.MessageIDPrefix)
	m.CreatedAt = s.now()
	s.messages[m.ID] = m
	s.mu.Unlock()
	s.afterWrite()
	return m, nil
}

func (s *MemoryStore) GetMessage(ctx context.Context, id string) (model.Message, error) {
	if err := ctx.Err(); err != nil {
		return model.Message{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.messages[id]
	if !ok {
		return model.Message{}, model.ErrNotFound
	}
	return m, nil
}

func (s *MemoryStore) ListMessagesByPost(ctx context.Context, postID string) ([]model.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Message, 0)
	for _, m := range s.messages {
		if m.PostID == postID {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) DeleteMessage(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	if _, ok := s.messages[id]; !ok {
		s.mu.Unlock()
		return model.ErrNotFound
	}
	delete(s.messages, id)
	s.mu.Unlock()
	s.afterWrite()
	return nil
}

func (s *MemoryStore) CreateFavorite(ctx context.Context, f model.Favorite) (model.Favorite, error) {
	if err := ctx.Err(); err != nil {
		return model.Favorite{}, err
	}
	s.mu.Lock()
	key := model.FavoriteKey{UserID: f.UserID, PostID: f.PostID}
	if _, ok := s.favIdx[key]; ok {
		s.mu.Unlock()
		return model.Favorite{}, model.ErrDuplicateFavorite
	}
	if _, ok := s.posts[f.PostID]; !ok {
		s.mu.Unlock()
		return model.Favorite{}, model.ErrNotFound
	}
	f.ID = s.genID(model.FavoriteIDPrefix)
	f.CreatedAt = s.now()
	s.favorites[f.ID] = f
	s.favIdx[key] = f.ID
	s.mu.Unlock()
	s.afterWrite()
	return f, nil
}

func (s *MemoryStore) DeleteFavorite(ctx context.Context, userID, postID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	key := model.FavoriteKey{UserID: userID, PostID: postID}
	id, ok := s.favIdx[key]
	if !ok {
		s.mu.Unlock()
		return model.ErrNotFound
	}
	delete(s.favorites, id)
	delete(s.favIdx, key)
	s.mu.Unlock()
	s.afterWrite()
	return nil
}

func (s *MemoryStore) GetFavorite(ctx context.Context, userID, postID string) (model.Favorite, error) {
	if err := ctx.Err(); err != nil {
		return model.Favorite{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.favIdx[model.FavoriteKey{UserID: userID, PostID: postID}]
	if !ok {
		return model.Favorite{}, model.ErrNotFound
	}
	return s.favorites[id], nil
}

func (s *MemoryStore) ListFavoritesByUser(ctx context.Context, userID string) ([]model.Favorite, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Favorite, 0)
	for _, f := range s.favorites {
		if f.UserID == userID {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) CreateNotification(ctx context.Context, n model.Notification) (model.Notification, error) {
	if err := ctx.Err(); err != nil {
		return model.Notification{}, err
	}
	s.mu.Lock()
	n.ID = s.genID(model.NotificationIDPrefix)
	n.CreatedAt = s.now()
	s.notifs[n.ID] = n
	s.mu.Unlock()
	s.afterWrite()
	return n, nil
}

func (s *MemoryStore) ListNotifications(ctx context.Context, userID string, unreadOnly bool) ([]model.Notification, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Notification, 0)
	for _, n := range s.notifs {
		if n.UserID != userID {
			continue
		}
		if unreadOnly && n.Read {
			continue
		}
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) GetNotification(ctx context.Context, id string) (model.Notification, error) {
	if err := ctx.Err(); err != nil {
		return model.Notification{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notifs[id]
	if !ok {
		return model.Notification{}, model.ErrNotFound
	}
	return n, nil
}

func (s *MemoryStore) UpdateNotification(ctx context.Context, n model.Notification) (model.Notification, error) {
	if err := ctx.Err(); err != nil {
		return model.Notification{}, err
	}
	s.mu.Lock()
	if _, ok := s.notifs[n.ID]; !ok {
		s.mu.Unlock()
		return model.Notification{}, model.ErrNotFound
	}
	s.notifs[n.ID] = n
	s.mu.Unlock()
	s.afterWrite()
	return n, nil
}

func (s *MemoryStore) MarkAllNotificationsRead(ctx context.Context, userID string, at time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	n := 0
	for id, item := range s.notifs {
		if item.UserID == userID && !item.Read {
			item.Read = true
			t := at
			item.ReadAt = &t
			s.notifs[id] = item
			n++
		}
	}
	s.mu.Unlock()
	if n > 0 {
		s.afterWrite()
	}
	return n, nil
}

func (s *MemoryStore) CountUnreadNotifications(ctx context.Context, userID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, item := range s.notifs {
		if item.UserID == userID && !item.Read {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) CreateReport(ctx context.Context, r model.Report) (model.Report, error) {
	if err := ctx.Err(); err != nil {
		return model.Report{}, err
	}
	s.mu.Lock()
	for _, old := range s.reports {
		if old.ReporterID == r.ReporterID && old.TargetType == r.TargetType && old.TargetID == r.TargetID && old.Status == model.ReportPending {
			s.mu.Unlock()
			return model.Report{}, model.ErrReportAlreadyOpen
		}
	}
	now := s.now()
	r.ID = s.genID(model.ReportIDPrefix)
	r.Status = model.ReportPending
	r.CreatedAt = now
	s.reports[r.ID] = r
	s.mu.Unlock()
	s.afterWrite()
	return r, nil
}

func (s *MemoryStore) GetReport(ctx context.Context, id string) (model.Report, error) {
	if err := ctx.Err(); err != nil {
		return model.Report{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.reports[id]
	if !ok {
		return model.Report{}, model.ErrNotFound
	}
	return r, nil
}

func (s *MemoryStore) ListReports(ctx context.Context, status model.ReportStatus) ([]model.Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Report, 0)
	for _, r := range s.reports {
		if status != "" && r.Status != status {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) UpdateReport(ctx context.Context, r model.Report) (model.Report, error) {
	if err := ctx.Err(); err != nil {
		return model.Report{}, err
	}
	s.mu.Lock()
	if _, ok := s.reports[r.ID]; !ok {
		s.mu.Unlock()
		return model.Report{}, model.ErrNotFound
	}
	s.reports[r.ID] = r
	s.mu.Unlock()
	s.afterWrite()
	return r, nil
}

func (s *MemoryStore) HasOpenReport(ctx context.Context, reporterID string, targetType model.ReportTarget, targetID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.reports {
		if r.ReporterID == reporterID && r.TargetType == targetType && r.TargetID == targetID && r.Status == model.ReportPending {
			return true, nil
		}
	}
	return false, nil
}

func (s *MemoryStore) CountPendingReports(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, r := range s.reports {
		if r.Status == model.ReportPending {
			n++
		}
	}
	return n, nil
}
