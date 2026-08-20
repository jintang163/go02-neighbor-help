package service

import (
	"context"
	"time"

	"go02-neighbor-help/internal/auth"
	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

var DefaultClock Clock = systemClock{}

type Services struct {
	Auth    *AuthService
	User    *UserService
	Post    *PostService
	Match   *MatchService
	Task    *TaskService
	Review  *ReviewService
	Social  *SocialService
	Report  *ReportService
	Stats   *StatsService
	Notify  *NotifyService
	MaxOpen int
}

func NewServices(s store.Store, hasher *auth.PasswordHasher, sessions *auth.SessionManager, clock Clock, maxOpen int) *Services {
	if clock == nil {
		clock = DefaultClock
	}
	if maxOpen <= 0 {
		maxOpen = 10
	}
	notify := NewNotifyService(s, clock)
	credit := NewCreditHelper(s, notify, clock)
	svc := &Services{
		Notify:  notify,
		MaxOpen: maxOpen,
	}
	svc.Auth = NewAuthService(s, hasher, sessions, clock, notify)
	svc.User = NewUserService(s, hasher, sessions, credit, clock)
	svc.Post = NewPostService(s, notify, clock, maxOpen)
	svc.Match = NewMatchService(s, notify, clock)
	svc.Task = NewTaskService(s, notify, credit, clock)
	svc.Review = NewReviewService(s, notify, credit, clock)
	svc.Social = NewSocialService(s, notify, clock)
	svc.Report = NewReportService(s, sessions, credit, notify, clock)
	svc.Stats = NewStatsService(s, clock)
	return svc
}

type ctxKey string

const ctxUserKey ctxKey = "user"

func WithUser(ctx context.Context, u model.User) context.Context {
	return context.WithValue(ctx, ctxUserKey, u)
}

func UserFromContext(ctx context.Context) (model.User, bool) {
	u, ok := ctx.Value(ctxUserKey).(model.User)
	return u, ok
}

func MustUserFromContext(ctx context.Context) model.User {
	u, ok := UserFromContext(ctx)
	if !ok {
		panic("service: user not found in context")
	}
	return u
}

func requireActiveWriter(u model.User) error {
	if u.IsAdmin() {
		return nil
	}
	return u.CanWrite()
}

func publicOf(ctx context.Context, s store.Store, id string) (model.PublicUser, error) {
	u, err := s.GetUserByID(ctx, id)
	if err != nil {
		return model.PublicUser{}, err
	}
	return u.Public(), nil
}
