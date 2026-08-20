package store

import (
	"context"
	"sort"
	"strings"

	"go02-neighbor-help/internal/model"
)

func (s *MemoryStore) CreateUser(ctx context.Context, u model.User) (model.User, error) {
	if err := ctx.Err(); err != nil {
		return model.User{}, err
	}
	u.Username = strings.TrimSpace(u.Username)
	u.DisplayName = sanitizeDisplayName(u.DisplayName)
	if u.Status == "" {
		u.Status = model.UserActive
	}
	if u.CreditScore == 0 && u.ID == "" {
		u.CreditScore = model.CreditInitial
	}
	s.mu.Lock()
	if _, exists := s.username[u.Username]; exists {
		s.mu.Unlock()
		return model.User{}, model.ErrAlreadyExists
	}
	now := s.now()
	u.ID = s.genID(model.UserIDPrefix)
	u.CreatedAt = now
	u.UpdatedAt = now
	s.users[u.ID] = u
	s.username[u.Username] = u.ID
	s.mu.Unlock()
	s.afterWrite()
	return u, nil
}

func (s *MemoryStore) GetUserByUsername(ctx context.Context, username string) (model.User, error) {
	if err := ctx.Err(); err != nil {
		return model.User{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.username[username]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return s.users[id], nil
}

func (s *MemoryStore) GetUserByID(ctx context.Context, id string) (model.User, error) {
	if err := ctx.Err(); err != nil {
		return model.User{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return u, nil
}

func (s *MemoryStore) ListUsers(ctx context.Context, f model.UserFilter) ([]model.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.User, 0, len(s.users))
	for _, u := range s.users {
		if f.Role != "" && u.Role != f.Role {
			continue
		}
		if f.Status != "" && u.Status.Normalize() != f.Status {
			continue
		}
		if f.Query != "" {
			blob := u.Username + " " + u.DisplayName + " " + u.Building
			if !matchQuery(blob, f.Query) {
				continue
			}
		}
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *MemoryStore) UpdateUser(ctx context.Context, u model.User) (model.User, error) {
	if err := ctx.Err(); err != nil {
		return model.User{}, err
	}
	s.mu.Lock()
	old, ok := s.users[u.ID]
	if !ok {
		s.mu.Unlock()
		return model.User{}, model.ErrNotFound
	}
	u.Username = old.Username
	u.CreatedAt = old.CreatedAt
	u.DisplayName = sanitizeDisplayName(u.DisplayName)
	u.UpdatedAt = s.now()
	s.users[u.ID] = u
	s.mu.Unlock()
	s.afterWrite()
	return u, nil
}

func (s *MemoryStore) CountUsers(ctx context.Context) (total, active, frozen int, err error) {
	if err = ctx.Err(); err != nil {
		return 0, 0, 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		total++
		switch u.Status.Normalize() {
		case model.UserActive:
			active++
		case model.UserFrozen:
			frozen++
		}
	}
	return total, active, frozen, nil
}

func (s *MemoryStore) ApplyCredit(ctx context.Context, userID string, delta int, reason model.CreditReason, relatedID, note string) (model.User, model.CreditLog, error) {
	if err := ctx.Err(); err != nil {
		return model.User{}, model.CreditLog{}, err
	}
	s.mu.Lock()
	u, ok := s.users[userID]
	if !ok {
		s.mu.Unlock()
		return model.User{}, model.CreditLog{}, model.ErrNotFound
	}
	u.CreditScore = model.ClampCredit(u.CreditScore + delta)
	u.UpdatedAt = s.now()
	s.users[userID] = u
	log := model.CreditLog{
		ID:         s.genID(model.CreditLogIDPrefix),
		UserID:     userID,
		Delta:      delta,
		Reason:     reason,
		RelatedID:  relatedID,
		ScoreAfter: u.CreditScore,
		Note:       note,
		CreatedAt:  s.now(),
	}
	s.credits[log.ID] = log
	s.mu.Unlock()
	s.afterWrite()
	return u, log, nil
}

func (s *MemoryStore) CreateCreditLog(ctx context.Context, l model.CreditLog) (model.CreditLog, error) {
	if err := ctx.Err(); err != nil {
		return model.CreditLog{}, err
	}
	s.mu.Lock()
	l.ID = s.genID(model.CreditLogIDPrefix)
	l.CreatedAt = s.now()
	s.credits[l.ID] = l
	s.mu.Unlock()
	s.afterWrite()
	return l, nil
}

func (s *MemoryStore) ListCreditLogs(ctx context.Context, userID string) ([]model.CreditLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.CreditLog, 0)
	for _, l := range s.credits {
		if l.UserID == userID {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
