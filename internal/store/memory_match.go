package store

import (
	"context"
	"sort"
	"time"

	"go02-neighbor-help/internal/model"
)

func (s *MemoryStore) CreateApplication(ctx context.Context, a model.Application) (model.Application, error) {
	if err := ctx.Err(); err != nil {
		return model.Application{}, err
	}
	s.mu.Lock()
	key := appKey(a.PostID, a.ApplicantID)
	if existingID, ok := s.appIdx[key]; ok {
		old := s.apps[existingID]
		if old.Status == model.AppPending || old.Status == model.AppAccepted {
			s.mu.Unlock()
			return model.Application{}, model.ErrAlreadyApplied
		}
	}
	now := s.now()
	a.ID = s.genID(model.ApplicationIDPrefix)
	a.Status = model.AppPending
	a.CreatedAt = now
	a.UpdatedAt = now
	s.apps[a.ID] = a
	s.appIdx[key] = a.ID
	if p, ok := s.posts[a.PostID]; ok {
		p.ApplyCount++
		p.UpdatedAt = now
		s.posts[a.PostID] = p
	}
	s.mu.Unlock()
	s.afterWrite()
	return a, nil
}

func (s *MemoryStore) GetApplication(ctx context.Context, id string) (model.Application, error) {
	if err := ctx.Err(); err != nil {
		return model.Application{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.apps[id]
	if !ok {
		return model.Application{}, model.ErrNotFound
	}
	return a, nil
}

func (s *MemoryStore) GetApplicationByPostApplicant(ctx context.Context, postID, applicantID string) (model.Application, error) {
	if err := ctx.Err(); err != nil {
		return model.Application{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.appIdx[appKey(postID, applicantID)]
	if !ok {
		return model.Application{}, model.ErrNotFound
	}
	return s.apps[id], nil
}

func (s *MemoryStore) ListApplicationsByPost(ctx context.Context, postID string) ([]model.Application, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Application, 0)
	for _, a := range s.apps {
		if a.PostID == postID {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) ListApplicationsByApplicant(ctx context.Context, applicantID string) ([]model.Application, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Application, 0)
	for _, a := range s.apps {
		if a.ApplicantID == applicantID {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) UpdateApplication(ctx context.Context, a model.Application) (model.Application, error) {
	if err := ctx.Err(); err != nil {
		return model.Application{}, err
	}
	s.mu.Lock()
	old, ok := s.apps[a.ID]
	if !ok {
		s.mu.Unlock()
		return model.Application{}, model.ErrNotFound
	}
	a.CreatedAt = old.CreatedAt
	a.PostID = old.PostID
	a.ApplicantID = old.ApplicantID
	a.UpdatedAt = s.now()
	s.apps[a.ID] = a
	s.mu.Unlock()
	s.afterWrite()
	return a, nil
}

func (s *MemoryStore) CreateTask(ctx context.Context, t model.Task) (model.Task, error) {
	if err := ctx.Err(); err != nil {
		return model.Task{}, err
	}
	s.mu.Lock()
	if _, ok := s.taskByPost[t.PostID]; ok {
		s.mu.Unlock()
		return model.Task{}, model.ErrAlreadyMatched
	}
	now := s.now()
	t.ID = s.genID(model.TaskIDPrefix)
	if t.Status == "" {
		t.Status = model.TaskPendingStart
	}
	t.CreatedAt = now
	t.UpdatedAt = now
	s.tasks[t.ID] = t
	s.taskByPost[t.PostID] = t.ID
	s.mu.Unlock()
	s.afterWrite()
	return t, nil
}

func (s *MemoryStore) GetTask(ctx context.Context, id string) (model.Task, error) {
	if err := ctx.Err(); err != nil {
		return model.Task{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return model.Task{}, model.ErrNotFound
	}
	return t, nil
}

func (s *MemoryStore) GetTaskByPost(ctx context.Context, postID string) (model.Task, error) {
	if err := ctx.Err(); err != nil {
		return model.Task{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.taskByPost[postID]
	if !ok {
		return model.Task{}, model.ErrNotFound
	}
	return s.tasks[id], nil
}

func (s *MemoryStore) ListTasksByUser(ctx context.Context, userID string) ([]model.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Task, 0)
	for _, t := range s.tasks {
		if t.IsParty(userID) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) UpdateTask(ctx context.Context, t model.Task) (model.Task, error) {
	if err := ctx.Err(); err != nil {
		return model.Task{}, err
	}
	s.mu.Lock()
	old, ok := s.tasks[t.ID]
	if !ok {
		s.mu.Unlock()
		return model.Task{}, model.ErrNotFound
	}
	t.CreatedAt = old.CreatedAt
	t.PostID = old.PostID
	t.RequesterID = old.RequesterID
	t.HelperID = old.HelperID
	t.UpdatedAt = s.now()
	s.tasks[t.ID] = t
	s.mu.Unlock()
	s.afterWrite()
	return t, nil
}

func (s *MemoryStore) CountActiveTasksByUser(ctx context.Context, userID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, t := range s.tasks {
		if t.IsParty(userID) && t.Status.IsActive() {
			n++
		}
	}
	return n, nil
}

func (s *MemoryStore) CountTasks(ctx context.Context) (total, completed, disputed int, err error) {
	if err = ctx.Err(); err != nil {
		return 0, 0, 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tasks {
		total++
		switch t.Status {
		case model.TaskCompleted:
			completed++
		case model.TaskDisputed:
			disputed++
		}
	}
	return total, completed, disputed, nil
}

func (s *MemoryStore) CountTasksCreatedOn(ctx context.Context, day time.Time) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, t := range s.tasks {
		if model.ClockDateEqual(t.CreatedAt, day) {
			n++
		}
	}
	return n, nil
}

// ConfirmTaskStart 在同一把写锁内推进“确认开始”，避免两个参与方并发确认时
// 各自基于旧副本回写而互相覆盖。仅 pending_start 允许确认；任一方重复确认无副作用。
// 返回更新后的任务，以及本次调用是否将任务推进到 in_progress。
func (s *MemoryStore) ConfirmTaskStart(ctx context.Context, taskID, actorID string, asAdmin bool) (model.Task, bool, error) {
	if err := ctx.Err(); err != nil {
		return model.Task{}, false, err
	}
	s.mu.Lock()
	task, ok := s.tasks[taskID]
	if !ok {
		s.mu.Unlock()
		return model.Task{}, false, model.ErrNotFound
	}
	// 锁内再次校验状态，避免在 GetTask 与本调用之间被取消/推进。
	if task.Status != model.TaskPendingStart {
		s.mu.Unlock()
		return model.Task{}, false, model.ErrInvalidTaskStatus
	}
	if !asAdmin && !task.IsParty(actorID) {
		s.mu.Unlock()
		return model.Task{}, false, model.ErrNotTaskParty
	}

	now := s.now()
	if asAdmin || task.RequesterID == actorID {
		task.RequesterStarted = true
	}
	if asAdmin || task.HelperID == actorID {
		task.HelperStarted = true
	}

	activated := false
	if task.RequesterStarted && task.HelperStarted {
		task.Status = model.TaskInProgress
		task.StartAt = &now
		activated = true
		if p, ok := s.posts[task.PostID]; ok {
			p.Status = model.PostInProgress
			p.UpdatedAt = now
			s.posts[task.PostID] = p
		}
	}
	task.UpdatedAt = now
	s.tasks[task.ID] = task
	s.mu.Unlock()
	s.afterWrite()
	return task, activated, nil
}

// AcceptMatch 同一把锁内完成匹配，避免双开。
func (s *MemoryStore) AcceptMatch(ctx context.Context, appID, actorID string, asAdmin bool) (model.Application, model.Task, model.HelpPost, error) {
	if err := ctx.Err(); err != nil {
		return model.Application{}, model.Task{}, model.HelpPost{}, err
	}
	s.mu.Lock()
	app, ok := s.apps[appID]
	if !ok {
		s.mu.Unlock()
		return model.Application{}, model.Task{}, model.HelpPost{}, model.ErrNotFound
	}
	if app.Status != model.AppPending {
		s.mu.Unlock()
		return model.Application{}, model.Task{}, model.HelpPost{}, model.ErrConflict
	}
	post, ok := s.posts[app.PostID]
	if !ok {
		s.mu.Unlock()
		return model.Application{}, model.Task{}, model.HelpPost{}, model.ErrNotFound
	}
	if post.Status != model.PostOpen {
		s.mu.Unlock()
		return model.Application{}, model.Task{}, model.HelpPost{}, model.ErrPostNotOpen
	}
	if !asAdmin && post.AuthorID != actorID {
		s.mu.Unlock()
		return model.Application{}, model.Task{}, model.HelpPost{}, model.ErrNotPostAuthor
	}
	if _, exists := s.taskByPost[post.ID]; exists {
		s.mu.Unlock()
		return model.Application{}, model.Task{}, model.HelpPost{}, model.ErrAlreadyMatched
	}

	now := s.now()
	var requesterID, helperID string
	switch post.Type {
	case model.PostRequest:
		requesterID, helperID = post.AuthorID, app.ApplicantID
	case model.PostOffer:
		helperID, requesterID = post.AuthorID, app.ApplicantID
	default:
		s.mu.Unlock()
		return model.Application{}, model.Task{}, model.HelpPost{}, model.ErrInvalidPostType
	}

	task := model.Task{
		ID:          s.genID(model.TaskIDPrefix),
		PostID:      post.ID,
		RequesterID: requesterID,
		HelperID:    helperID,
		Status:      model.TaskPendingStart,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.tasks[task.ID] = task
	s.taskByPost[post.ID] = task.ID

	app.Status = model.AppAccepted
	app.UpdatedAt = now
	tnow := now
	app.DecidedAt = &tnow
	s.apps[app.ID] = app

	for id, other := range s.apps {
		if other.PostID == post.ID && other.ID != app.ID && other.Status == model.AppPending {
			other.Status = model.AppRejected
			other.UpdatedAt = now
			other.DecidedAt = &tnow
			s.apps[id] = other
		}
	}

	post.Status = model.PostMatched
	post.MatchedUserID = app.ApplicantID
	post.TaskID = task.ID
	post.UpdatedAt = now
	s.posts[post.ID] = post

	s.mu.Unlock()
	s.afterWrite()
	return app, task, post, nil
}
