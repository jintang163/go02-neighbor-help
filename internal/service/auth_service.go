package service

import (
	"context"
	"strings"

	"go02-neighbor-help/internal/auth"
	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type AuthService struct {
	store    store.Store
	hasher   *auth.PasswordHasher
	sessions *auth.SessionManager
	clock    Clock
	notify   *NotifyService
}

func NewAuthService(s store.Store, hasher *auth.PasswordHasher, sessions *auth.SessionManager, clock Clock, notify *NotifyService) *AuthService {
	return &AuthService{store: s, hasher: hasher, sessions: sessions, clock: clock, notify: notify}
}

func (a *AuthService) Register(ctx context.Context, in model.UserInput) (model.AuthResponse, error) {
	in.Normalize()
	in.Role = model.RoleResident
	if err := in.Validate(); err != nil {
		return model.AuthResponse{}, err
	}
	return a.createAndLogin(ctx, in)
}

func (a *AuthService) Login(ctx context.Context, in model.LoginInput) (model.AuthResponse, error) {
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || in.Password == "" {
		return model.AuthResponse{}, model.ErrInvalidCredentials
	}
	u, err := a.store.GetUserByUsername(ctx, in.Username)
	if err != nil {
		if model.IsNotFound(err) {
			return model.AuthResponse{}, model.ErrInvalidCredentials
		}
		return model.AuthResponse{}, err
	}
	if u.IsBanned() {
		return model.AuthResponse{}, model.ErrAccountBanned
	}
	if !a.hasher.Verify(in.Password, u.PasswordSalt, u.PasswordHash, u.Iterations) {
		return model.AuthResponse{}, model.ErrInvalidCredentials
	}
	now := a.clock.Now()
	u.LastLoginAt = &now
	u, err = a.store.UpdateUser(ctx, u)
	if err != nil {
		return model.AuthResponse{}, err
	}
	token, err := a.sessions.Create(u)
	if err != nil {
		return model.AuthResponse{}, err
	}
	return model.AuthResponse{Token: token, User: u.Public()}, nil
}

func (a *AuthService) Logout(token string) {
	a.sessions.Invalidate(token)
}

func (a *AuthService) Me(ctx context.Context, u model.User) (model.MeResponse, error) {
	fresh, err := a.store.GetUserByID(ctx, u.ID)
	if err != nil {
		return model.MeResponse{}, err
	}
	unread, _ := a.store.CountUnreadNotifications(ctx, fresh.ID)
	tasks, _ := a.store.ListTasksByUser(ctx, fresh.ID)
	pendingStart, pendingConfirm, pendingReview := 0, 0, 0
	for _, t := range tasks {
		switch t.Status {
		case model.TaskPendingStart:
			pendingStart++
		case model.TaskPendingConfirm:
			if t.RequesterID == fresh.ID {
				pendingConfirm++
			}
		case model.TaskCompleted:
			if t.RoleOf(fresh.ID) == "requester" && !t.RequesterReviewed {
				pendingReview++
			}
			if t.RoleOf(fresh.ID) == "helper" && !t.HelperReviewed {
				pendingReview++
			}
		}
	}
	openPosts, _ := a.store.CountActivePostsByAuthor(ctx, fresh.ID)
	apps, _ := a.store.ListApplicationsByApplicant(ctx, fresh.ID)
	pendingApply := 0
	for _, ap := range apps {
		if ap.Status == model.AppPending {
			pendingApply++
		}
	}
	return model.MeResponse{
		User:           fresh.Public(),
		UnreadNotify:   unread,
		PendingStart:   pendingStart,
		PendingConfirm: pendingConfirm,
		PendingReview:  pendingReview,
		OpenPosts:      openPosts,
		PendingApply:   pendingApply,
	}, nil
}

func (a *AuthService) ChangePassword(ctx context.Context, u model.User, in model.PasswordInput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	fresh, err := a.store.GetUserByID(ctx, u.ID)
	if err != nil {
		return err
	}
	if !a.hasher.Verify(in.OldPassword, fresh.PasswordSalt, fresh.PasswordHash, fresh.Iterations) {
		return model.ErrInvalidCredentials
	}
	salt, hash, it, err := a.hasher.Hash(in.NewPassword)
	if err != nil {
		return err
	}
	fresh.PasswordSalt = salt
	fresh.PasswordHash = hash
	fresh.Iterations = it
	if _, err := a.store.UpdateUser(ctx, fresh); err != nil {
		return err
	}
	a.sessions.InvalidateByUser(fresh.ID)
	return nil
}

func (a *AuthService) createAndLogin(ctx context.Context, in model.UserInput) (model.AuthResponse, error) {
	salt, hash, it, err := a.hasher.Hash(in.Password)
	if err != nil {
		return model.AuthResponse{}, err
	}
	u, err := a.store.CreateUser(ctx, model.User{
		Username:     in.Username,
		PasswordHash: hash,
		PasswordSalt: salt,
		Iterations:   it,
		Role:         in.Role,
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
		return model.AuthResponse{}, err
	}
	token, err := a.sessions.Create(u)
	if err != nil {
		return model.AuthResponse{}, err
	}
	if a.notify != nil {
		a.notify.Push(ctx, u.ID, model.NotifySystem, "欢迎加入邻里互助", "发布求助或提供帮助，完成后记得互相评价。", u.ID, "user")
	}
	return model.AuthResponse{Token: token, User: u.Public()}, nil
}
