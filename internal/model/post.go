package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// PostType 帖子类型：求助或提供帮助。
type PostType string

const (
	PostRequest PostType = "request"
	PostOffer   PostType = "offer"
)

func (t PostType) IsValid() bool {
	return t == PostRequest || t == PostOffer
}

// PostStatus 互助帖状态。
type PostStatus string

const (
	PostDraft          PostStatus = "draft"
	PostOpen           PostStatus = "open"
	PostMatched        PostStatus = "matched"
	PostInProgress     PostStatus = "in_progress"
	PostPendingConfirm PostStatus = "pending_confirm"
	PostCompleted      PostStatus = "completed"
	PostCancelled      PostStatus = "cancelled"
	PostExpired        PostStatus = "expired"
	PostClosed         PostStatus = "closed"
)

func (s PostStatus) IsValid() bool {
	switch s {
	case PostDraft, PostOpen, PostMatched, PostInProgress, PostPendingConfirm,
		PostCompleted, PostCancelled, PostExpired, PostClosed:
		return true
	}
	return false
}

// IsActive 仍占用“进行中配额”的状态。
func (s PostStatus) IsActive() bool {
	switch s {
	case PostDraft, PostOpen, PostMatched, PostInProgress, PostPendingConfirm:
		return true
	}
	return false
}

// IsTerminal 终态。
func (s PostStatus) IsTerminal() bool {
	switch s {
	case PostCompleted, PostCancelled, PostExpired, PostClosed:
		return true
	}
	return false
}

// VisibleToPlaza 是否出现在广场默认列表。
func (s PostStatus) VisibleToPlaza() bool {
	return s == PostOpen
}

// HelpPost 互助帖。
type HelpPost struct {
	ID             string     `json:"id"`
	Type           PostType   `json:"type"`
	Status         PostStatus `json:"status"`
	Category       Category   `json:"category"`
	Urgency        Urgency    `json:"urgency"`
	Title          string     `json:"title"`
	Content        string     `json:"content"`
	Building       string     `json:"building,omitempty"`
	LocationNote   string     `json:"location_note,omitempty"`
	TimeWindowStart *time.Time `json:"time_window_start,omitempty"`
	TimeWindowEnd   *time.Time `json:"time_window_end,omitempty"`
	RewardType     RewardType `json:"reward_type"`
	RewardNote     string     `json:"reward_note,omitempty"`
	AuthorID       string     `json:"author_id"`
	MatchedUserID  string     `json:"matched_user_id,omitempty"`
	TaskID         string     `json:"task_id,omitempty"`
	ViewCount      int        `json:"view_count"`
	ApplyCount     int        `json:"apply_count"`
	ClosedReason   string     `json:"closed_reason,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
}

// IsExpiredAt 在给定时刻是否超过时间窗口且仍为开放。
func (p HelpPost) IsExpiredAt(now time.Time) bool {
	if p.Status != PostOpen {
		return false
	}
	if p.TimeWindowEnd == nil {
		return false
	}
	return now.After(*p.TimeWindowEnd)
}

func (p HelpPost) VisibleTo(user User) bool {
	if user.IsAdmin() {
		return true
	}
	if p.AuthorID == user.ID {
		return true
	}
	if p.MatchedUserID == user.ID {
		return true
	}
	return p.Status != PostDraft
}

type PostInput struct {
	Type            PostType   `json:"type"`
	Category        Category   `json:"category"`
	Urgency         Urgency    `json:"urgency"`
	Title           string     `json:"title"`
	Content         string     `json:"content"`
	Building        string     `json:"building"`
	LocationNote    string     `json:"location_note"`
	TimeWindowStart *time.Time `json:"time_window_start"`
	TimeWindowEnd   *time.Time `json:"time_window_end"`
	RewardType      RewardType `json:"reward_type"`
	RewardNote      string     `json:"reward_note"`
	Publish         bool       `json:"publish"`
}

func (in *PostInput) Normalize() {
	in.Title = strings.TrimSpace(in.Title)
	in.Content = strings.TrimSpace(in.Content)
	in.Building = strings.TrimSpace(in.Building)
	in.LocationNote = strings.TrimSpace(in.LocationNote)
	in.RewardNote = strings.TrimSpace(in.RewardNote)
	if in.Urgency == "" {
		in.Urgency = UrgencyNormal
	}
	if in.Category == "" {
		in.Category = CategoryOther
	}
	in.RewardType = in.RewardType.Normalize()
}

func (in PostInput) Validate() error {
	in.Normalize()
	if !in.Type.IsValid() {
		return ErrInvalidPostType
	}
	if !in.Category.IsValid() {
		return ErrInvalidCategory
	}
	if !in.Urgency.IsValid() {
		return ErrInvalidUrgency
	}
	if !in.RewardType.IsValid() {
		return ErrInvalidReward
	}
	tn := utf8.RuneCountInString(in.Title)
	if tn < 1 || tn > 80 {
		return ErrInvalidTitle
	}
	cn := utf8.RuneCountInString(in.Content)
	if cn < 1 || cn > 4000 {
		return ErrInvalidContent
	}
	if utf8.RuneCountInString(in.Building) > 16 {
		return ErrInvalidLocation
	}
	if utf8.RuneCountInString(in.LocationNote) > 80 {
		return ErrInvalidLocation
	}
	if utf8.RuneCountInString(in.RewardNote) > 80 {
		return ErrInvalidComment
	}
	if in.TimeWindowStart != nil && in.TimeWindowEnd != nil && !in.TimeWindowEnd.After(*in.TimeWindowStart) {
		return ErrInvalidTimeWindow
	}
	return nil
}

type PostFilter struct {
	Type     PostType
	Status   PostStatus
	Category Category
	Urgency  Urgency
	Building string
	AuthorID string
	Query    string
	Plaza    bool // 广场：默认只看 open
}

type PostListItem struct {
	HelpPost
	Author       PublicUser `json:"author"`
	MatchedUser  *PublicUser `json:"matched_user,omitempty"`
	Favorited    bool        `json:"favorited"`
}
