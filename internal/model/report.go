package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// ReportTarget 举报对象类型。
type ReportTarget string

const (
	ReportUser    ReportTarget = "user"
	ReportPost    ReportTarget = "post"
	ReportReview  ReportTarget = "review"
	ReportMessage ReportTarget = "message"
)

func (t ReportTarget) IsValid() bool {
	switch t {
	case ReportUser, ReportPost, ReportReview, ReportMessage:
		return true
	}
	return false
}

// ReportStatus 举报处理状态。
type ReportStatus string

const (
	ReportPending  ReportStatus = "pending"
	ReportAccepted ReportStatus = "accepted"
	ReportRejected ReportStatus = "rejected"
)

func (s ReportStatus) IsValid() bool {
	switch s {
	case ReportPending, ReportAccepted, ReportRejected:
		return true
	}
	return false
}

// Report 举报。
type Report struct {
	ID         string       `json:"id"`
	ReporterID string       `json:"reporter_id"`
	TargetType ReportTarget `json:"target_type"`
	TargetID   string       `json:"target_id"`
	Reason     string       `json:"reason"`
	Detail     string       `json:"detail,omitempty"`
	Status     ReportStatus `json:"status"`
	HandlerID  string       `json:"handler_id,omitempty"`
	HandleNote string       `json:"handle_note,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	HandledAt  *time.Time   `json:"handled_at,omitempty"`
}

type ReportInput struct {
	TargetType ReportTarget `json:"target_type"`
	TargetID   string       `json:"target_id"`
	Reason     string       `json:"reason"`
	Detail     string       `json:"detail"`
}

func (in *ReportInput) Normalize() {
	in.TargetID = strings.TrimSpace(in.TargetID)
	in.Reason = strings.TrimSpace(in.Reason)
	in.Detail = strings.TrimSpace(in.Detail)
}

func (in ReportInput) Validate() error {
	in.Normalize()
	if !in.TargetType.IsValid() {
		return ErrValidation
	}
	if in.TargetID == "" {
		return ErrValidation
	}
	n := utf8.RuneCountInString(in.Reason)
	if n < 1 || n > 80 {
		return ErrInvalidComment
	}
	if utf8.RuneCountInString(in.Detail) > 500 {
		return ErrInvalidComment
	}
	return nil
}

type HandleReportInput struct {
	Action string `json:"action"` // accept / reject
	Note   string `json:"note"`
	Freeze bool   `json:"freeze"`
}

func (in *HandleReportInput) Normalize() {
	in.Action = strings.TrimSpace(strings.ToLower(in.Action))
	in.Note = strings.TrimSpace(in.Note)
}

func (in HandleReportInput) Validate() error {
	in.Normalize()
	if in.Action != "accept" && in.Action != "reject" {
		return ErrInvalidHandleAction
	}
	if utf8.RuneCountInString(in.Note) > 500 {
		return ErrInvalidComment
	}
	return nil
}

type ReportView struct {
	Report
	Reporter PublicUser `json:"reporter"`
}

// CreditReason 信用流水原因。
type CreditReason string

const (
	CreditCompleteHelper    CreditReason = "complete_helper"
	CreditCompleteRequester CreditReason = "complete_requester"
	CreditReviewReceived    CreditReason = "review_received"
	CreditCancelAfterStart  CreditReason = "cancel_after_start"
	CreditReportAccepted    CreditReason = "report_accepted"
	CreditAdminAdjust       CreditReason = "admin_adjust"
)

// CreditLog 信用流水。
type CreditLog struct {
	ID         string       `json:"id"`
	UserID     string       `json:"user_id"`
	Delta      int          `json:"delta"`
	Reason     CreditReason `json:"reason"`
	RelatedID  string       `json:"related_id,omitempty"`
	ScoreAfter int          `json:"score_after"`
	Note       string       `json:"note,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
}

type CreditAdjustInput struct {
	Delta int    `json:"delta"`
	Note  string `json:"note"`
}

func (in *CreditAdjustInput) Normalize() {
	in.Note = strings.TrimSpace(in.Note)
}

func (in CreditAdjustInput) Validate() error {
	in.Normalize()
	if in.Delta < -20 || in.Delta > 20 || in.Delta == 0 {
		return ErrInvalidCreditDelta
	}
	if utf8.RuneCountInString(in.Note) > 200 {
		return ErrInvalidComment
	}
	return nil
}
