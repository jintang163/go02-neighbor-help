package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// TaskStatus 互助任务状态。
type TaskStatus string

const (
	TaskPendingStart    TaskStatus = "pending_start"
	TaskInProgress      TaskStatus = "in_progress"
	TaskPendingConfirm  TaskStatus = "pending_confirm"
	TaskCompleted       TaskStatus = "completed"
	TaskCancelled       TaskStatus = "cancelled"
	TaskDisputed        TaskStatus = "disputed"
)

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskPendingStart, TaskInProgress, TaskPendingConfirm, TaskCompleted, TaskCancelled, TaskDisputed:
		return true
	}
	return false
}

func (s TaskStatus) IsActive() bool {
	switch s {
	case TaskPendingStart, TaskInProgress, TaskPendingConfirm, TaskDisputed:
		return true
	}
	return false
}

func (s TaskStatus) IsTerminal() bool {
	return s == TaskCompleted || s == TaskCancelled
}

// Task 匹配成功后的履约任务。始终同时记录 requester 与 helper。
type Task struct {
	ID                  string     `json:"id"`
	PostID              string     `json:"post_id"`
	RequesterID         string     `json:"requester_id"`
	HelperID            string     `json:"helper_id"`
	Status              TaskStatus `json:"status"`
	RequesterStarted    bool       `json:"requester_started"`
	HelperStarted       bool       `json:"helper_started"`
	StartAt             *time.Time `json:"start_at,omitempty"`
	CompleteAt          *time.Time `json:"complete_at,omitempty"`
	CancelReason        string     `json:"cancel_reason,omitempty"`
	DisputeReason       string     `json:"dispute_reason,omitempty"`
	CancelledBy         string     `json:"cancelled_by,omitempty"`
	BothReviewed        bool       `json:"both_reviewed"`
	RequesterReviewed   bool       `json:"requester_reviewed"`
	HelperReviewed      bool       `json:"helper_reviewed"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (t Task) IsParty(userID string) bool {
	return t.RequesterID == userID || t.HelperID == userID
}

func (t Task) RoleOf(userID string) string {
	switch userID {
	case t.RequesterID:
		return "requester"
	case t.HelperID:
		return "helper"
	default:
		return ""
	}
}

func (t Task) Counterpart(userID string) string {
	switch userID {
	case t.RequesterID:
		return t.HelperID
	case t.HelperID:
		return t.RequesterID
	default:
		return ""
	}
}

type CancelInput struct {
	Reason string `json:"reason"`
}

func (in *CancelInput) Normalize() {
	in.Reason = strings.TrimSpace(in.Reason)
}

func (in CancelInput) Validate() error {
	in.Normalize()
	n := utf8.RuneCountInString(in.Reason)
	if n > 500 {
		return ErrInvalidComment
	}
	return nil
}

type DisputeInput struct {
	Reason string `json:"reason"`
}

func (in *DisputeInput) Normalize() {
	in.Reason = strings.TrimSpace(in.Reason)
}

func (in DisputeInput) Validate() error {
	in.Normalize()
	n := utf8.RuneCountInString(in.Reason)
	if n < 1 || n > 500 {
		return ErrInvalidComment
	}
	return nil
}

type TaskView struct {
	Task
	Post      HelpPost    `json:"post"`
	Requester PublicUser  `json:"requester"`
	Helper    PublicUser  `json:"helper"`
}
