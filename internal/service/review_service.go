package service

import (
	"context"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type ReviewService struct {
	store  store.Store
	notify *NotifyService
	credit *CreditHelper
	clock  Clock
}

func NewReviewService(s store.Store, notify *NotifyService, credit *CreditHelper, clock Clock) *ReviewService {
	return &ReviewService{store: s, notify: notify, credit: credit, clock: clock}
}

func (r *ReviewService) Submit(ctx context.Context, actor model.User, taskID string, in model.ReviewInput) (model.Review, error) {
	if err := requireActiveWriter(actor); err != nil {
		return model.Review{}, err
	}
	in.Normalize()
	if err := in.Validate(); err != nil {
		return model.Review{}, err
	}
	task, err := r.store.GetTask(ctx, taskID)
	if err != nil {
		return model.Review{}, err
	}
	if task.Status != model.TaskCompleted {
		return model.Review{}, model.ErrTaskNotCompleted
	}
	if !task.IsParty(actor.ID) {
		return model.Review{}, model.ErrNotTaskParty
	}
	if _, err := r.store.GetReviewByTaskFrom(ctx, task.ID, actor.ID); err == nil {
		return model.Review{}, model.ErrAlreadyReviewed
	} else if !model.IsNotFound(err) {
		return model.Review{}, err
	}
	role := model.ReviewAsRequester
	if actor.ID == task.HelperID {
		role = model.ReviewAsHelper
	}
	created, err := r.store.CreateReview(ctx, model.Review{
		TaskID:     task.ID,
		PostID:     task.PostID,
		FromUserID: actor.ID,
		ToUserID:   task.Counterpart(actor.ID),
		Role:       role,
		Score:      in.Score,
		Tags:       in.Tags,
		Comment:    in.Comment,
		Anonymous:  in.Anonymous,
	})
	if err != nil {
		return model.Review{}, err
	}
	// 信用账本写入是评价流程中最易失败的一步。若它失败但评价记录已落库，
	// (task, from) 槽位会被占用，导致重试被 ErrAlreadyReviewed 拒绝，而信用分
	// 实际并未变化——卡在不可重试的半成功状态。这里删除刚创建的评价记录以
	// 释放槽位并回滚评分副作用，使整体可安全重试。信用写入本身失败，无需冲销。
	delta := model.CreditDeltaForScore(in.Score)
	if _, err := r.credit.Apply(ctx, created.ToUserID, delta, model.CreditReviewReceived, created.ID, "收到评价"); err != nil {
		_ = r.store.DeleteReview(ctx, created.ID)
		return model.Review{}, err
	}
	r.notify.Push(ctx, created.ToUserID, model.NotifyReviewed, "收到新评价", actor.DisplayName+" 给你打了 "+itoa(in.Score)+" 星", created.ID, "review")
	return created, nil
}

func (r *ReviewService) ListByTask(ctx context.Context, actor model.User, taskID string) ([]model.ReviewView, error) {
	task, err := r.store.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if !task.IsParty(actor.ID) && !actor.IsAdmin() {
		return nil, model.ErrNotTaskParty
	}
	items, err := r.store.ListReviewsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return r.toViews(ctx, items), nil
}

func (r *ReviewService) toViews(ctx context.Context, items []model.Review) []model.ReviewView {
	out := make([]model.ReviewView, 0, len(items))
	for _, item := range items {
		view := model.ReviewView{Review: item}
		if from, err := r.store.GetUserByID(ctx, item.FromUserID); err == nil {
			if item.Anonymous {
				from.DisplayName = "匿名邻居"
				from.Username = "anonymous"
			}
			view.FromUser = from.Public()
		}
		if to, err := r.store.GetUserByID(ctx, item.ToUserID); err == nil {
			view.ToUser = to.Public()
		}
		out = append(out, view)
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
