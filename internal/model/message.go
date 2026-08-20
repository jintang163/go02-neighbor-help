package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Message 帖子留言。
type Message struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	AuthorID  string    `json:"author_id"`
	ParentID  string    `json:"parent_id,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type MessageInput struct {
	Content  string `json:"content"`
	ParentID string `json:"parent_id"`
}

func (in *MessageInput) Normalize() {
	in.Content = strings.TrimSpace(in.Content)
	in.ParentID = strings.TrimSpace(in.ParentID)
}

func (in MessageInput) Validate() error {
	in.Normalize()
	n := utf8.RuneCountInString(in.Content)
	if n < 1 || n > 500 {
		return ErrInvalidComment
	}
	return nil
}

type MessageView struct {
	Message
	Author PublicUser `json:"author"`
}

// Favorite 收藏。
type Favorite struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	PostID    string    `json:"post_id"`
	CreatedAt time.Time `json:"created_at"`
}

type FavoriteKey struct {
	UserID string
	PostID string
}

// NotificationType 站内通知类型。
type NotificationType string

const (
	NotifyApplied         NotificationType = "applied"
	NotifyAccepted        NotificationType = "accepted"
	NotifyRejected        NotificationType = "rejected"
	NotifyWithdrawn       NotificationType = "withdrawn"
	NotifyTaskStarted     NotificationType = "task_started"
	NotifyTaskCompleted   NotificationType = "task_completed"
	NotifyTaskCancelled   NotificationType = "task_cancelled"
	NotifyTaskDisputed    NotificationType = "task_disputed"
	NotifyReviewed        NotificationType = "reviewed"
	NotifyReportHandled   NotificationType = "report_handled"
	NotifyCreditChanged   NotificationType = "credit_changed"
	NotifyPostExpired     NotificationType = "post_expired"
	NotifySystem          NotificationType = "system"
)

// Notification 站内通知。
type Notification struct {
	ID          string           `json:"id"`
	UserID      string           `json:"user_id"`
	Type        NotificationType `json:"type"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	RelatedID   string           `json:"related_id,omitempty"`
	RelatedType string           `json:"related_type,omitempty"`
	Read        bool             `json:"read"`
	ReadAt      *time.Time       `json:"read_at,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
}

type NotificationFilter struct {
	UnreadOnly bool
	Limit      int
}
