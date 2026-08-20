package service

import (
	"context"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type SocialService struct {
	store  store.Store
	notify *NotifyService
	clock  Clock
}

func NewSocialService(s store.Store, notify *NotifyService, clock Clock) *SocialService {
	return &SocialService{store: s, notify: notify, clock: clock}
}

func (s *SocialService) AddMessage(ctx context.Context, actor model.User, postID string, in model.MessageInput) (model.MessageView, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.MessageView{}, err
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.MessageView{}, err
	}
	post, err := s.store.GetPost(ctx, postID)
	if err != nil {
		return model.MessageView{}, err
	}
	if !post.VisibleTo(actor) {
		return model.MessageView{}, model.ErrNotFound
	}
	msg, err := s.store.CreateMessage(ctx, model.Message{
		PostID:   postID,
		AuthorID: actor.ID,
		ParentID: in.ParentID,
		Content:  in.Content,
	})
	if err != nil {
		return model.MessageView{}, err
	}
	if post.AuthorID != actor.ID {
		s.notify.Push(ctx, post.AuthorID, model.NotifySystem, "帖子有新留言", actor.DisplayName+" 留言了《"+post.Title+"》", post.ID, "post")
	}
	return model.MessageView{Message: msg, Author: actor.Public()}, nil
}

func (s *SocialService) ListMessages(ctx context.Context, actor model.User, postID string) ([]model.MessageView, error) {
	post, err := s.store.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if !post.VisibleTo(actor) {
		return nil, model.ErrNotFound
	}
	items, err := s.store.ListMessagesByPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	out := make([]model.MessageView, 0, len(items))
	for _, m := range items {
		view := model.MessageView{Message: m}
		if u, err := publicOf(ctx, s.store, m.AuthorID); err == nil {
			view.Author = u
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *SocialService) DeleteMessage(ctx context.Context, actor model.User, id string) error {
	msg, err := s.store.GetMessage(ctx, id)
	if err != nil {
		return err
	}
	if msg.AuthorID != actor.ID && !actor.IsAdmin() {
		return model.ErrForbidden
	}
	return s.store.DeleteMessage(ctx, id)
}

func (s *SocialService) Favorite(ctx context.Context, actor model.User, postID string) (model.Favorite, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Favorite{}, err
	}
	return s.store.CreateFavorite(ctx, model.Favorite{UserID: actor.ID, PostID: postID})
}

func (s *SocialService) Unfavorite(ctx context.Context, actor model.User, postID string) error {
	return s.store.DeleteFavorite(ctx, actor.ID, postID)
}

func (s *SocialService) MyFavorites(ctx context.Context, actor model.User) ([]model.PostListItem, error) {
	favs, err := s.store.ListFavoritesByUser(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	out := make([]model.PostListItem, 0, len(favs))
	for _, f := range favs {
		post, err := s.store.GetPost(ctx, f.PostID)
		if err != nil {
			continue
		}
		author, err := publicOf(ctx, s.store, post.AuthorID)
		if err != nil {
			continue
		}
		out = append(out, model.PostListItem{HelpPost: post, Author: author, Favorited: true})
	}
	return out, nil
}
