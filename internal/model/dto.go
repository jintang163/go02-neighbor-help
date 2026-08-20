package model

import "time"

// ErrorResponse JSON 错误体。
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HealthResponse 健康检查。
type HealthResponse struct {
	Status string `json:"status"`
}

// LoginInput 登录请求。
type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// PasswordInput 改密请求。
type PasswordInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (in PasswordInput) Validate() error {
	if !IsValidPassword(in.NewPassword) {
		return ErrInvalidPassword
	}
	if in.OldPassword == "" {
		return ErrInvalidPassword
	}
	return nil
}

// AuthResponse 登录成功响应。
type AuthResponse struct {
	Token string     `json:"token"`
	User  PublicUser `json:"user"`
}

// MeResponse 当前用户 + 待办摘要。
type MeResponse struct {
	User              PublicUser `json:"user"`
	UnreadNotify      int        `json:"unread_notify"`
	PendingStart      int        `json:"pending_start"`
	PendingConfirm    int        `json:"pending_confirm"`
	PendingReview     int        `json:"pending_review"`
	OpenPosts         int        `json:"open_posts"`
	PendingApply      int        `json:"pending_apply"`
}

// GlobalStats 管理员全局统计。
type GlobalStats struct {
	UserTotal       int     `json:"user_total"`
	UserActive      int     `json:"user_active"`
	UserFrozen      int     `json:"user_frozen"`
	PostTotal       int     `json:"post_total"`
	PostOpen        int     `json:"post_open"`
	PostMatched     int     `json:"post_matched"`
	PostInProgress  int     `json:"post_in_progress"`
	PostCompleted   int     `json:"post_completed"`
	PostCancelled   int     `json:"post_cancelled"`
	TaskTotal       int     `json:"task_total"`
	TaskCompleted   int     `json:"task_completed"`
	TaskDisputed    int     `json:"task_disputed"`
	CompleteRate    float64 `json:"complete_rate"`
	AvgReviewScore  float64 `json:"avg_review_score"`
	ReviewTotal     int     `json:"review_total"`
	PendingReports  int     `json:"pending_reports"`
	TodayNewPosts   int     `json:"today_new_posts"`
	TodayNewTasks   int     `json:"today_new_tasks"`
}

// ListResponse 通用列表包装。
type ListResponse[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

// DictResponse 前端字典。
type DictResponse struct {
	Categories     []CategoryInfo `json:"categories"`
	PositiveTags   []string       `json:"positive_tags"`
	NegativeTags   []string       `json:"negative_tags"`
	NeutralTags    []string       `json:"neutral_tags"`
	UrgencyOptions []string       `json:"urgency_options"`
	RewardOptions  []string       `json:"reward_options"`
}

func DefaultDict() DictResponse {
	return DictResponse{
		Categories:     AllCategories(),
		PositiveTags:   PositiveReviewTags(),
		NegativeTags:   NegativeReviewTags(),
		NeutralTags:    NeutralReviewTags(),
		UrgencyOptions: []string{string(UrgencyLow), string(UrgencyNormal), string(UrgencyHigh), string(UrgencyUrgent)},
		RewardOptions:  []string{string(RewardNone), string(RewardThanks), string(RewardPoints), string(RewardInKind)},
	}
}

// NewList 构造列表响应。
func NewList[T any](items []T) ListResponse[T] {
	if items == nil {
		items = []T{}
	}
	return ListResponse[T]{Items: items, Total: len(items)}
}

// ClockDateEqual 判断两个时间是否同一天（按本地时区）。
func ClockDateEqual(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
