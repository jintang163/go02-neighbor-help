package service

import (
	"context"
	"fmt"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type NotifyService struct {
	store store.Store
	clock Clock
}

func NewNotifyService(s store.Store, clock Clock) *NotifyService {
	return &NotifyService{store: s, clock: clock}
}

func (n *NotifyService) Push(ctx context.Context, userID string, typ model.NotificationType, title, body, relatedID, relatedType string) {
	if userID == "" {
		return
	}
	_, _ = n.store.CreateNotification(ctx, model.Notification{
		UserID:      userID,
		Type:        typ,
		Title:       title,
		Body:        body,
		RelatedID:   relatedID,
		RelatedType: relatedType,
	})
}

func (n *NotifyService) List(ctx context.Context, userID string, unreadOnly bool) ([]model.Notification, error) {
	return n.store.ListNotifications(ctx, userID, unreadOnly)
}

func (n *NotifyService) MarkRead(ctx context.Context, user model.User, id string) (model.Notification, error) {
	item, err := n.store.GetNotification(ctx, id)
	if err != nil {
		return model.Notification{}, err
	}
	if item.UserID != user.ID && !user.IsAdmin() {
		return model.Notification{}, model.ErrForbidden
	}
	if item.Read {
		return item, nil
	}
	now := n.clock.Now()
	item.Read = true
	item.ReadAt = &now
	return n.store.UpdateNotification(ctx, item)
}

func (n *NotifyService) MarkAllRead(ctx context.Context, userID string) (int, error) {
	return n.store.MarkAllNotificationsRead(ctx, userID, n.clock.Now())
}

func (n *NotifyService) UnreadCount(ctx context.Context, userID string) (int, error) {
	return n.store.CountUnreadNotifications(ctx, userID)
}

type CreditHelper struct {
	store  store.Store
	notify *NotifyService
	clock  Clock
}

func NewCreditHelper(s store.Store, notify *NotifyService, clock Clock) *CreditHelper {
	return &CreditHelper{store: s, notify: notify, clock: clock}
}

func (c *CreditHelper) Apply(ctx context.Context, userID string, delta int, reason model.CreditReason, relatedID, note string) (model.User, error) {
	if delta == 0 {
		return c.store.GetUserByID(ctx, userID)
	}
	u, _, err := c.store.ApplyCredit(ctx, userID, delta, reason, relatedID, note)
	if err != nil {
		return model.User{}, err
	}
	if c.notify != nil && delta != 0 {
		sign := "+"
		if delta < 0 {
			sign = ""
		}
		c.notify.Push(ctx, userID, model.NotifyCreditChanged,
			"信用分变更",
			fmt.Sprintf("信用分 %s%d，当前 %d（%s）", sign, delta, u.CreditScore, u.CreditLevel()),
			relatedID, "credit")
	}
	return u, nil
}
