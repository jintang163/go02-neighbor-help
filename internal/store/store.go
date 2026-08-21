package store

import (
	"context"
	"time"

	"go02-neighbor-help/internal/model"
)

// Store 数据访问接口。service 层只依赖本接口。
type Store interface {
	CreateUser(ctx context.Context, u model.User) (model.User, error)
	GetUserByUsername(ctx context.Context, username string) (model.User, error)
	GetUserByID(ctx context.Context, id string) (model.User, error)
	ListUsers(ctx context.Context, f model.UserFilter) ([]model.User, error)
	UpdateUser(ctx context.Context, u model.User) (model.User, error)
	CountUsers(ctx context.Context) (total, active, frozen int, err error)

	CreatePost(ctx context.Context, p model.HelpPost) (model.HelpPost, error)
	GetPost(ctx context.Context, id string) (model.HelpPost, error)
	ListPosts(ctx context.Context, f model.PostFilter) ([]model.HelpPost, error)
	UpdatePost(ctx context.Context, p model.HelpPost) (model.HelpPost, error)
	CountActivePostsByAuthor(ctx context.Context, authorID string) (int, error)
	CountPostsByStatus(ctx context.Context) (map[model.PostStatus]int, int, error)
	CountPostsCreatedOn(ctx context.Context, day time.Time) (int, error)

	CreateApplication(ctx context.Context, a model.Application) (model.Application, error)
	GetApplication(ctx context.Context, id string) (model.Application, error)
	GetApplicationByPostApplicant(ctx context.Context, postID, applicantID string) (model.Application, error)
	ListApplicationsByPost(ctx context.Context, postID string) ([]model.Application, error)
	ListApplicationsByApplicant(ctx context.Context, applicantID string) ([]model.Application, error)
	UpdateApplication(ctx context.Context, a model.Application) (model.Application, error)

	CreateTask(ctx context.Context, t model.Task) (model.Task, error)
	GetTask(ctx context.Context, id string) (model.Task, error)
	GetTaskByPost(ctx context.Context, postID string) (model.Task, error)
	ListTasksByUser(ctx context.Context, userID string) ([]model.Task, error)
	UpdateTask(ctx context.Context, t model.Task) (model.Task, error)
	// ConfirmTaskStart 在同一把写锁内完成“确认开始”的状态推进，避免两个参与方
	// 并发确认时各自基于旧副本回写而互相覆盖。asAdmin 为 true 时同时确认双方。
	// 第二个返回值表示本次调用是否把任务推进到了 in_progress。
	ConfirmTaskStart(ctx context.Context, taskID, actorID string, asAdmin bool) (model.Task, bool, error)
	CountActiveTasksByUser(ctx context.Context, userID string) (int, error)
	CountTasks(ctx context.Context) (total, completed, disputed int, err error)
	CountTasksCreatedOn(ctx context.Context, day time.Time) (int, error)

	CreateReview(ctx context.Context, r model.Review) (model.Review, error)
	GetReview(ctx context.Context, id string) (model.Review, error)
	GetReviewByTaskFrom(ctx context.Context, taskID, fromUserID string) (model.Review, error)
	ListReviewsByTask(ctx context.Context, taskID string) ([]model.Review, error)
	ListReviewsToUser(ctx context.Context, userID string) ([]model.Review, error)
	UpdateReview(ctx context.Context, r model.Review) (model.Review, error)
	CountReviews(ctx context.Context) (count int, sum int, err error)

	CreateMessage(ctx context.Context, m model.Message) (model.Message, error)
	GetMessage(ctx context.Context, id string) (model.Message, error)
	ListMessagesByPost(ctx context.Context, postID string) ([]model.Message, error)
	DeleteMessage(ctx context.Context, id string) error

	CreateFavorite(ctx context.Context, f model.Favorite) (model.Favorite, error)
	DeleteFavorite(ctx context.Context, userID, postID string) error
	GetFavorite(ctx context.Context, userID, postID string) (model.Favorite, error)
	ListFavoritesByUser(ctx context.Context, userID string) ([]model.Favorite, error)

	CreateNotification(ctx context.Context, n model.Notification) (model.Notification, error)
	ListNotifications(ctx context.Context, userID string, unreadOnly bool) ([]model.Notification, error)
	GetNotification(ctx context.Context, id string) (model.Notification, error)
	UpdateNotification(ctx context.Context, n model.Notification) (model.Notification, error)
	MarkAllNotificationsRead(ctx context.Context, userID string, at time.Time) (int, error)
	CountUnreadNotifications(ctx context.Context, userID string) (int, error)

	CreateReport(ctx context.Context, r model.Report) (model.Report, error)
	GetReport(ctx context.Context, id string) (model.Report, error)
	ListReports(ctx context.Context, status model.ReportStatus) ([]model.Report, error)
	UpdateReport(ctx context.Context, r model.Report) (model.Report, error)
	HasOpenReport(ctx context.Context, reporterID string, targetType model.ReportTarget, targetID string) (bool, error)
	CountPendingReports(ctx context.Context) (int, error)

	CreateCreditLog(ctx context.Context, l model.CreditLog) (model.CreditLog, error)
	ListCreditLogs(ctx context.Context, userID string) ([]model.CreditLog, error)

	AcceptMatch(ctx context.Context, appID, actorID string, asAdmin bool) (model.Application, model.Task, model.HelpPost, error)
	ExpireOpenPost(ctx context.Context, postID string) (model.HelpPost, error)
	ApplyCredit(ctx context.Context, userID string, delta int, reason model.CreditReason, relatedID, note string) (model.User, model.CreditLog, error)
}

var _ Store = (*MemoryStore)(nil)
