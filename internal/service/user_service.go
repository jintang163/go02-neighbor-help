package service

import (
	"context"

	"go02-neighbor-help/internal/auth"
	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type UserService struct {
	store    store.Store
	hasher   *auth.PasswordHasher
	sessions *auth.SessionManager
	credit   *CreditHelper
	clock    Clock
}

func NewUserService(s store.Store, hasher *auth.PasswordHasher, sessions *auth.SessionManager, credit *CreditHelper, clock Clock) *UserService {
	return &UserService{store: s, hasher: hasher, sessions: sessions, credit: credit, clock: clock}
}

func (u *UserService) GetByID(ctx context.Context, id string) (model.User, error) {
	return u.store.GetUserByID(ctx, id)
}

func (u *UserService) PublicProfile(ctx context.Context, id string) (model.PublicUser, error) {
	user, err := u.store.GetUserByID(ctx, id)
	if err != nil {
		return model.PublicUser{}, err
	}
	return user.Public(), nil
}

func (u *UserService) List(ctx context.Context, f model.UserFilter) ([]model.PublicUser, error) {
	users, err := u.store.ListUsers(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]model.PublicUser, 0, len(users))
	for _, item := range users {
		out = append(out, item.Public())
	}
	return out, nil
}

func (u *UserService) CreateResident(ctx context.Context, actor model.User, in model.UserInput) (model.PublicUser, error) {
	if !actor.IsAdmin() {
		return model.PublicUser{}, model.ErrForbidden
	}
	in.Normalize()
	in.Role = model.RoleResident
	if err := in.Validate(); err != nil {
		return model.PublicUser{}, err
	}
	salt, hash, it, err := u.hasher.Hash(in.Password)
	if err != nil {
		return model.PublicUser{}, err
	}
	created, err := u.store.CreateUser(ctx, model.User{
		Username:     in.Username,
		PasswordHash: hash,
		PasswordSalt: salt,
		Iterations:   it,
		Role:         model.RoleResident,
		Status:       model.UserActive,
		DisplayName:  in.DisplayName,
		Building:     in.Building,
		Unit:         in.Unit,
		Room:         in.Room,
		Phone:        in.Phone,
		Bio:          in.Bio,
		CreditScore:  model.CreditInitial,
	})
	if err != nil {
		return model.PublicUser{}, err
	}
	return created.Public(), nil
}

func (u *UserService) UpdateProfile(ctx context.Context, actor model.User, in model.ProfileInput) (model.PublicUser, error) {
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.PublicUser{}, err
	}
	fresh, err := u.store.GetUserByID(ctx, actor.ID)
	if err != nil {
		return model.PublicUser{}, err
	}
	fresh.DisplayName = in.DisplayName
	fresh.Building = in.Building
	fresh.Unit = in.Unit
	fresh.Room = in.Room
	fresh.Phone = in.Phone
	fresh.Bio = in.Bio
	updated, err := u.store.UpdateUser(ctx, fresh)
	if err != nil {
		return model.PublicUser{}, err
	}
	return updated.Public(), nil
}

func (u *UserService) Freeze(ctx context.Context, actor model.User, targetID string) (model.PublicUser, error) {
	return u.setStatus(ctx, actor, targetID, model.UserFrozen)
}

func (u *UserService) Unfreeze(ctx context.Context, actor model.User, targetID string) (model.PublicUser, error) {
	return u.setStatus(ctx, actor, targetID, model.UserActive)
}

func (u *UserService) setStatus(ctx context.Context, actor model.User, targetID string, status model.UserStatus) (model.PublicUser, error) {
	if !actor.IsAdmin() {
		return model.PublicUser{}, model.ErrForbidden
	}
	target, err := u.store.GetUserByID(ctx, targetID)
	if err != nil {
		return model.PublicUser{}, err
	}
	if target.IsAdmin() {
		return model.PublicUser{}, model.ErrForbidden
	}
	target.Status = status
	updated, err := u.store.UpdateUser(ctx, target)
	if err != nil {
		return model.PublicUser{}, err
	}
	if status != model.UserActive {
		u.sessions.InvalidateByUser(targetID)
	}
	return updated.Public(), nil
}

func (u *UserService) AdjustCredit(ctx context.Context, actor model.User, targetID string, in model.CreditAdjustInput) (model.PublicUser, error) {
	if !actor.IsAdmin() {
		return model.PublicUser{}, model.ErrForbidden
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.PublicUser{}, err
	}
	updated, err := u.credit.Apply(ctx, targetID, in.Delta, model.CreditAdminAdjust, actor.ID, in.Note)
	if err != nil {
		return model.PublicUser{}, err
	}
	return updated.Public(), nil
}

func (u *UserService) CreditLogs(ctx context.Context, actor model.User, userID string) ([]model.CreditLog, error) {
	if actor.ID != userID && !actor.IsAdmin() {
		return nil, model.ErrForbidden
	}
	return u.store.ListCreditLogs(ctx, userID)
}

func (u *UserService) ReviewsReceived(ctx context.Context, userID string) ([]model.ReviewView, error) {
	if _, err := u.store.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}
	items, err := u.store.ListReviewsToUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]model.ReviewView, 0, len(items))
	for _, r := range items {
		view := model.ReviewView{Review: r}
		if from, err := u.store.GetUserByID(ctx, r.FromUserID); err == nil {
			if r.Anonymous {
				from.DisplayName = "匿名邻居"
				from.Username = "anonymous"
			}
			view.FromUser = from.Public()
		}
		if to, err := u.store.GetUserByID(ctx, r.ToUserID); err == nil {
			view.ToUser = to.Public()
		}
		out = append(out, view)
	}
	return out, nil
}
