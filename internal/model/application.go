package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// ApplicationStatus 报名状态。
type ApplicationStatus string

const (
	AppPending   ApplicationStatus = "pending"
	AppAccepted  ApplicationStatus = "accepted"
	AppRejected  ApplicationStatus = "rejected"
	AppWithdrawn ApplicationStatus = "withdrawn"
	AppExpired   ApplicationStatus = "expired"
)

func (s ApplicationStatus) IsValid() bool {
	switch s {
	case AppPending, AppAccepted, AppRejected, AppWithdrawn, AppExpired:
		return true
	}
	return false
}

func (s ApplicationStatus) IsOpen() bool { return s == AppPending }

// Application 对互助帖的报名。
type Application struct {
	ID          string            `json:"id"`
	PostID      string            `json:"post_id"`
	ApplicantID string            `json:"applicant_id"`
	Message     string            `json:"message,omitempty"`
	Status      ApplicationStatus `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	DecidedAt   *time.Time        `json:"decided_at,omitempty"`
}

type ApplyInput struct {
	Message string `json:"message"`
}

func (in *ApplyInput) Normalize() {
	in.Message = strings.TrimSpace(in.Message)
}

func (in ApplyInput) Validate() error {
	in.Normalize()
	if utf8.RuneCountInString(in.Message) > 500 {
		return ErrInvalidComment
	}
	return nil
}

type ApplicationView struct {
	Application
	Applicant PublicUser `json:"applicant"`
}
