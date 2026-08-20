package service

import (
	"context"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type MatchService struct {
	store  store.Store
	notify *NotifyService
	clock  Clock
}

func NewMatchService(s store.Store, notify *NotifyService, clock Clock) *MatchService {
	return &MatchService{store: s, notify: notify, clock: clock}
}

func (m *MatchService) Apply(ctx context.Context, actor model.User, postID string, in model.ApplyInput) (model.Application, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Application{}, err
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Application{}, err
	}
	post, err := m.store.GetPost(ctx, postID)
	if err != nil {
		return model.Application{}, err
	}
	if post.IsExpiredAt(m.clock.Now()) {
		_, _ = m.store.ExpireOpenPost(ctx, post.ID)
		return model.Application{}, model.ErrPostExpired
	}
	if post.Status != model.PostOpen {
		return model.Application{}, model.ErrPostNotOpen
	}
	if post.AuthorID == actor.ID {
		return model.Application{}, model.ErrCannotApplyOwnPost
	}
	if actor.CreditLevel() == model.CreditRestricted && post.Urgency.Rank() >= model.UrgencyHigh.Rank() {
		return model.Application{}, model.ErrCreditRestricted
	}
	if actor.CreditLevel() == model.CreditRestricted {
		n, err := m.store.CountActiveTasksByUser(ctx, actor.ID)
		if err != nil {
			return model.Application{}, err
		}
		if n >= 1 {
			return model.Application{}, model.ErrCreditRestricted
		}
	}
	if post.Urgency.RequiresMinCredit() > actor.CreditScore {
		return model.Application{}, model.ErrCreditRestricted
	}
	app, err := m.store.CreateApplication(ctx, model.Application{
		PostID:      post.ID,
		ApplicantID: actor.ID,
		Message:     in.Message,
	})
	if err != nil {
		return model.Application{}, err
	}
	m.notify.Push(ctx, post.AuthorID, model.NotifyApplied,
		"收到新的报名", actor.DisplayName+" 报名了《"+post.Title+"》", post.ID, "post")
	return app, nil
}

func (m *MatchService) Withdraw(ctx context.Context, actor model.User, appID string) (model.Application, error) {
	app, err := m.store.GetApplication(ctx, appID)
	if err != nil {
		return model.Application{}, err
	}
	if app.ApplicantID != actor.ID && !actor.IsAdmin() {
		return model.Application{}, model.ErrForbidden
	}
	if app.Status != model.AppPending {
		return model.Application{}, model.ErrConflict
	}
	app.Status = model.AppWithdrawn
	updated, err := m.store.UpdateApplication(ctx, app)
	if err != nil {
		return model.Application{}, err
	}
	if post, err := m.store.GetPost(ctx, app.PostID); err == nil {
		m.notify.Push(ctx, post.AuthorID, model.NotifyWithdrawn, "报名已撤回", actor.DisplayName+" 撤回了对《"+post.Title+"》的报名", post.ID, "post")
	}
	return updated, nil
}

func (m *MatchService) Accept(ctx context.Context, actor model.User, appID string) (model.TaskView, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.TaskView{}, err
	}
	app, task, post, err := m.store.AcceptMatch(ctx, appID, actor.ID, actor.IsAdmin())
	if err != nil {
		return model.TaskView{}, err
	}
	m.notify.Push(ctx, app.ApplicantID, model.NotifyAccepted, "报名已被接受", "《"+post.Title+"》已匹配成功，请确认开始。", task.ID, "task")
	m.notify.Push(ctx, post.AuthorID, model.NotifyAccepted, "已完成匹配", "《"+post.Title+"》已匹配成功，请确认开始。", task.ID, "task")
	rejected, _ := m.store.ListApplicationsByPost(ctx, post.ID)
	for _, other := range rejected {
		if other.Status == model.AppRejected {
			m.notify.Push(ctx, other.ApplicantID, model.NotifyRejected, "报名未通过", "《"+post.Title+"》已匹配其他人。", post.ID, "post")
		}
	}
	return m.taskView(ctx, task)
}

func (m *MatchService) Reject(ctx context.Context, actor model.User, appID string) (model.Application, error) {
	app, err := m.store.GetApplication(ctx, appID)
	if err != nil {
		return model.Application{}, err
	}
	post, err := m.store.GetPost(ctx, app.PostID)
	if err != nil {
		return model.Application{}, err
	}
	if post.AuthorID != actor.ID && !actor.IsAdmin() {
		return model.Application{}, model.ErrNotPostAuthor
	}
	if app.Status != model.AppPending {
		return model.Application{}, model.ErrConflict
	}
	app.Status = model.AppRejected
	now := m.clock.Now()
	app.DecidedAt = &now
	updated, err := m.store.UpdateApplication(ctx, app)
	if err != nil {
		return model.Application{}, err
	}
	m.notify.Push(ctx, app.ApplicantID, model.NotifyRejected, "报名未通过", "《"+post.Title+"》的作者未接受你的报名。", post.ID, "post")
	return updated, nil
}

func (m *MatchService) ListByPost(ctx context.Context, actor model.User, postID string) ([]model.ApplicationView, error) {
	post, err := m.store.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post.AuthorID != actor.ID && !actor.IsAdmin() {
		return nil, model.ErrNotPostAuthor
	}
	apps, err := m.store.ListApplicationsByPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	return m.toViews(ctx, apps)
}

func (m *MatchService) Mine(ctx context.Context, actor model.User) ([]model.ApplicationView, error) {
	apps, err := m.store.ListApplicationsByApplicant(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	return m.toViews(ctx, apps)
}

func (m *MatchService) toViews(ctx context.Context, apps []model.Application) ([]model.ApplicationView, error) {
	out := make([]model.ApplicationView, 0, len(apps))
	for _, a := range apps {
		view := model.ApplicationView{Application: a}
		if u, err := publicOf(ctx, m.store, a.ApplicantID); err == nil {
			view.Applicant = u
		}
		out = append(out, view)
	}
	return out, nil
}

func (m *MatchService) taskView(ctx context.Context, t model.Task) (model.TaskView, error) {
	post, err := m.store.GetPost(ctx, t.PostID)
	if err != nil {
		return model.TaskView{}, err
	}
	req, err := publicOf(ctx, m.store, t.RequesterID)
	if err != nil {
		return model.TaskView{}, err
	}
	help, err := publicOf(ctx, m.store, t.HelperID)
	if err != nil {
		return model.TaskView{}, err
	}
	return model.TaskView{Task: t, Post: post, Requester: req, Helper: help}, nil
}
