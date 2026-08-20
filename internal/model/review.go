package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// ReviewRole 评价人在任务中的角色。
type ReviewRole string

const (
	ReviewAsRequester ReviewRole = "as_requester"
	ReviewAsHelper    ReviewRole = "as_helper"
)

func (r ReviewRole) IsValid() bool {
	return r == ReviewAsRequester || r == ReviewAsHelper
}

var (
	positiveTags = map[string]struct{}{
		"punctual": {}, "kind": {}, "reliable": {}, "communicative": {}, "exceeded": {},
	}
	negativeTags = map[string]struct{}{
		"late": {}, "rude": {}, "unfinished": {}, "poor_comm": {}, "overstated": {},
	}
	neutralTags = map[string]struct{}{
		"average": {}, "improvable": {},
	}
)

// PositiveReviewTags 正向标签。
func PositiveReviewTags() []string {
	return []string{"punctual", "kind", "reliable", "communicative", "exceeded"}
}

// NegativeReviewTags 负向标签。
func NegativeReviewTags() []string {
	return []string{"late", "rude", "unfinished", "poor_comm", "overstated"}
}

// NeutralReviewTags 中性标签。
func NeutralReviewTags() []string {
	return []string{"average", "improvable"}
}

func tagKind(tag string) string {
	if _, ok := positiveTags[tag]; ok {
		return "positive"
	}
	if _, ok := negativeTags[tag]; ok {
		return "negative"
	}
	if _, ok := neutralTags[tag]; ok {
		return "neutral"
	}
	return ""
}

// Review 任务完成后的互评。
type Review struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"task_id"`
	PostID     string     `json:"post_id"`
	FromUserID string     `json:"from_user_id"`
	ToUserID   string     `json:"to_user_id"`
	Role       ReviewRole `json:"role"`
	Score      int        `json:"score"`
	Tags       []string   `json:"tags,omitempty"`
	Comment    string     `json:"comment,omitempty"`
	Anonymous  bool       `json:"anonymous"`
	Hidden     bool       `json:"hidden"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreditDeltaForScore 评价星级对应的被评人信用变化。
func CreditDeltaForScore(score int) int {
	switch score {
	case 5:
		return 3
	case 4:
		return 1
	case 3:
		return 0
	case 2:
		return -2
	case 1:
		return -5
	default:
		return 0
	}
}

type ReviewInput struct {
	Score     int      `json:"score"`
	Tags      []string `json:"tags"`
	Comment   string   `json:"comment"`
	Anonymous bool     `json:"anonymous"`
}

func (in *ReviewInput) Normalize() {
	in.Comment = strings.TrimSpace(in.Comment)
	clean := make([]string, 0, len(in.Tags))
	seen := map[string]struct{}{}
	for _, t := range in.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		clean = append(clean, t)
	}
	in.Tags = clean
}

func (in ReviewInput) Validate() error {
	in.Normalize()
	if in.Score < 1 || in.Score > 5 {
		return ErrInvalidScore
	}
	if utf8.RuneCountInString(in.Comment) > 500 {
		return ErrInvalidComment
	}
	if len(in.Tags) > 5 {
		return ErrInvalidReviewTags
	}
	for _, t := range in.Tags {
		kind := tagKind(t)
		if kind == "" {
			return ErrInvalidReviewTags
		}
		switch {
		case in.Score >= 4 && kind != "positive":
			return ErrInvalidReviewTags
		case in.Score <= 2 && kind != "negative":
			return ErrInvalidReviewTags
		case in.Score == 3 && kind != "neutral":
			return ErrInvalidReviewTags
		}
	}
	return nil
}

type ReviewView struct {
	Review
	FromUser PublicUser `json:"from_user"`
	ToUser   PublicUser `json:"to_user"`
}
