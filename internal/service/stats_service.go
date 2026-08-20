package service

import (
	"context"

	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/store"
)

type StatsService struct {
	store store.Store
	clock Clock
}

func NewStatsService(s store.Store, clock Clock) *StatsService {
	return &StatsService{store: s, clock: clock}
}

func (s *StatsService) Global(ctx context.Context, actor model.User) (model.GlobalStats, error) {
	if !actor.IsAdmin() {
		return model.GlobalStats{}, model.ErrForbidden
	}
	total, active, frozen, err := s.store.CountUsers(ctx)
	if err != nil {
		return model.GlobalStats{}, err
	}
	byStatus, postTotal, err := s.store.CountPostsByStatus(ctx)
	if err != nil {
		return model.GlobalStats{}, err
	}
	taskTotal, taskCompleted, taskDisputed, err := s.store.CountTasks(ctx)
	if err != nil {
		return model.GlobalStats{}, err
	}
	reviewCount, reviewSum, err := s.store.CountReviews(ctx)
	if err != nil {
		return model.GlobalStats{}, err
	}
	pendingReports, err := s.store.CountPendingReports(ctx)
	if err != nil {
		return model.GlobalStats{}, err
	}
	now := s.clock.Now()
	todayPosts, _ := s.store.CountPostsCreatedOn(ctx, now)
	todayTasks, _ := s.store.CountTasksCreatedOn(ctx, now)
	completeRate := 0.0
	if taskTotal > 0 {
		completeRate = float64(taskCompleted) / float64(taskTotal)
	}
	avg := 0.0
	if reviewCount > 0 {
		avg = float64(reviewSum) / float64(reviewCount)
	}
	return model.GlobalStats{
		UserTotal:      total,
		UserActive:     active,
		UserFrozen:     frozen,
		PostTotal:      postTotal,
		PostOpen:       byStatus[model.PostOpen],
		PostMatched:    byStatus[model.PostMatched],
		PostInProgress: byStatus[model.PostInProgress],
		PostCompleted:  byStatus[model.PostCompleted],
		PostCancelled:  byStatus[model.PostCancelled],
		TaskTotal:      taskTotal,
		TaskCompleted:  taskCompleted,
		TaskDisputed:   taskDisputed,
		CompleteRate:   completeRate,
		AvgReviewScore: avg,
		ReviewTotal:    reviewCount,
		PendingReports: pendingReports,
		TodayNewPosts:  todayPosts,
		TodayNewTasks:  todayTasks,
	}, nil
}
