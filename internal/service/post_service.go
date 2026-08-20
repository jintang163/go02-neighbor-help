package service

import (
	"context"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type PostService struct {
	store   store.Store
	notify  *NotifyService
	clock   Clock
	maxOpen int
}

func NewPostService(s store.Store, notify *NotifyService, clock Clock, maxOpen int) *PostService {
	return &PostService{store: s, notify: notify, clock: clock, maxOpen: maxOpen}
}

func (p *PostService) Create(ctx context.Context, actor model.User, in model.PostInput) (model.HelpPost, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.HelpPost{}, err
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.HelpPost{}, err
	}
	if actor.CreditScore < in.Urgency.RequiresMinCredit() {
		return model.HelpPost{}, model.ErrCreditRestricted
	}
	n, err := p.store.CountActivePostsByAuthor(ctx, actor.ID)
	if err != nil {
		return model.HelpPost{}, err
	}
	if n >= p.maxOpen {
		return model.HelpPost{}, model.ErrTooManyOpenPosts
	}
	post := model.HelpPost{
		Type:            in.Type,
		Status:          model.PostDraft,
		Category:        in.Category,
		Urgency:         in.Urgency,
		Title:           in.Title,
		Content:         in.Content,
		Building:        in.Building,
		LocationNote:    in.LocationNote,
		TimeWindowStart: in.TimeWindowStart,
		TimeWindowEnd:   in.TimeWindowEnd,
		RewardType:      in.RewardType,
		RewardNote:      in.RewardNote,
		AuthorID:        actor.ID,
	}
	created, err := p.store.CreatePost(ctx, post)
	if err != nil {
		return model.HelpPost{}, err
	}
	if in.Publish {
		return p.Publish(ctx, actor, created.ID)
	}
	return created, nil
}

func (p *PostService) Publish(ctx context.Context, actor model.User, id string) (model.HelpPost, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.HelpPost{}, err
	}
	post, err := p.mustOwnOrAdmin(ctx, actor, id)
	if err != nil {
		return model.HelpPost{}, err
	}
	if post.Status != model.PostDraft && post.Status != model.PostCancelled {
		return model.HelpPost{}, model.ErrConflict
	}
	if actor.CreditScore < post.Urgency.RequiresMinCredit() && !actor.IsAdmin() {
		return model.HelpPost{}, model.ErrCreditRestricted
	}
	now := p.clock.Now()
	if post.IsExpiredAt(now) {
		return model.HelpPost{}, model.ErrPostExpired
	}
	post.Status = model.PostOpen
	t := now
	post.PublishedAt = &t
	return p.store.UpdatePost(ctx, post)
}

func (p *PostService) Update(ctx context.Context, actor model.User, id string, in model.PostInput) (model.HelpPost, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.HelpPost{}, err
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.HelpPost{}, err
	}
	post, err := p.mustOwnOrAdmin(ctx, actor, id)
	if err != nil {
		return model.HelpPost{}, err
	}
	switch post.Status {
	case model.PostDraft:
		post.Type = in.Type
		post.Category = in.Category
		post.Urgency = in.Urgency
		post.Title = in.Title
		post.Content = in.Content
		post.Building = in.Building
		post.LocationNote = in.LocationNote
		post.TimeWindowStart = in.TimeWindowStart
		post.TimeWindowEnd = in.TimeWindowEnd
		post.RewardType = in.RewardType
		post.RewardNote = in.RewardNote
	case model.PostOpen:
		post.Title = in.Title
		post.Content = in.Content
		post.TimeWindowStart = in.TimeWindowStart
		post.TimeWindowEnd = in.TimeWindowEnd
		post.LocationNote = in.LocationNote
		post.RewardNote = in.RewardNote
	default:
		return model.HelpPost{}, model.ErrConflict
	}
	return p.store.UpdatePost(ctx, post)
}

func (p *PostService) Cancel(ctx context.Context, actor model.User, id string, reason string) (model.HelpPost, error) {
	post, err := p.mustOwnOrAdmin(ctx, actor, id)
	if err != nil {
		return model.HelpPost{}, err
	}
	if post.Status.IsTerminal() {
		return model.HelpPost{}, model.ErrConflict
	}
	if post.Status == model.PostInProgress || post.Status == model.PostPendingConfirm || post.Status == model.PostMatched {
		return model.HelpPost{}, model.ErrConflict
	}
	post.Status = model.PostCancelled
	post.ClosedReason = reason
	updated, err := p.store.UpdatePost(ctx, post)
	if err != nil {
		return model.HelpPost{}, err
	}
	apps, _ := p.store.ListApplicationsByPost(ctx, post.ID)
	for _, a := range apps {
		if a.Status == model.AppPending {
			a.Status = model.AppRejected
			_, _ = p.store.UpdateApplication(ctx, a)
			p.notify.Push(ctx, a.ApplicantID, model.NotifyRejected, "报名已失效", "作者取消了互助帖："+post.Title, post.ID, "post")
		}
	}
	return updated, nil
}

func (p *PostService) Get(ctx context.Context, actor model.User, id string) (model.PostListItem, error) {
	post, err := p.ensureFresh(ctx, id)
	if err != nil {
		return model.PostListItem{}, err
	}
	if !post.VisibleTo(actor) {
		return model.PostListItem{}, model.ErrNotFound
	}
	if actor.ID != post.AuthorID && !actor.IsAdmin() {
		post.ViewCount++
		post, _ = p.store.UpdatePost(ctx, post)
	}
	return p.toItem(ctx, actor, post)
}

func (p *PostService) List(ctx context.Context, actor model.User, f model.PostFilter) ([]model.PostListItem, error) {
	if f.Plaza && !actor.IsAdmin() {
		f.Status = model.PostOpen
	}
	if err := p.expireOpen(ctx); err != nil {
		return nil, err
	}
	posts, err := p.store.ListPosts(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]model.PostListItem, 0, len(posts))
	for _, post := range posts {
		if !post.VisibleTo(actor) {
			continue
		}
		item, err := p.toItem(ctx, actor, post)
		if err != nil {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (p *PostService) Mine(ctx context.Context, actor model.User) ([]model.PostListItem, error) {
	return p.List(ctx, actor, model.PostFilter{AuthorID: actor.ID, Plaza: false})
}

func (p *PostService) ForceClose(ctx context.Context, actor model.User, id, reason string) (model.HelpPost, error) {
	if !actor.IsAdmin() {
		return model.HelpPost{}, model.ErrForbidden
	}
	post, err := p.store.GetPost(ctx, id)
	if err != nil {
		return model.HelpPost{}, err
	}
	post.Status = model.PostClosed
	post.ClosedReason = reason
	updated, err := p.store.UpdatePost(ctx, post)
	if err != nil {
		return model.HelpPost{}, err
	}
	p.notify.Push(ctx, post.AuthorID, model.NotifySystem, "帖子已被管理员关闭", reason, post.ID, "post")
	return updated, nil
}

func (p *PostService) mustOwnOrAdmin(ctx context.Context, actor model.User, id string) (model.HelpPost, error) {
	post, err := p.store.GetPost(ctx, id)
	if err != nil {
		return model.HelpPost{}, err
	}
	if post.AuthorID != actor.ID && !actor.IsAdmin() {
		return model.HelpPost{}, model.ErrNotPostAuthor
	}
	return post, nil
}

func (p *PostService) ensureFresh(ctx context.Context, id string) (model.HelpPost, error) {
	post, err := p.store.GetPost(ctx, id)
	if err != nil {
		return model.HelpPost{}, err
	}
	if post.IsExpiredAt(p.clock.Now()) {
		return p.store.ExpireOpenPost(ctx, post.ID)
	}
	return post, nil
}

func (p *PostService) expireOpen(ctx context.Context) error {
	posts, err := p.store.ListPosts(ctx, model.PostFilter{Status: model.PostOpen})
	if err != nil {
		return err
	}
	now := p.clock.Now()
	for _, post := range posts {
		if post.IsExpiredAt(now) {
			expired, err := p.store.ExpireOpenPost(ctx, post.ID)
			if err == nil {
				p.notify.Push(ctx, expired.AuthorID, model.NotifyPostExpired, "互助帖已过期", expired.Title, expired.ID, "post")
			}
		}
	}
	return nil
}

func (p *PostService) toItem(ctx context.Context, actor model.User, post model.HelpPost) (model.PostListItem, error) {
	author, err := publicOf(ctx, p.store, post.AuthorID)
	if err != nil {
		return model.PostListItem{}, err
	}
	item := model.PostListItem{HelpPost: post, Author: author}
	if post.MatchedUserID != "" {
		if mu, err := publicOf(ctx, p.store, post.MatchedUserID); err == nil {
			item.MatchedUser = &mu
		}
	}
	if _, err := p.store.GetFavorite(ctx, actor.ID, post.ID); err == nil {
		item.Favorited = true
	}
	return item, nil
}
