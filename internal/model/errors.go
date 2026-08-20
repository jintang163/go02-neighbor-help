package model

import "errors"

// 领域错误。HTTP 层根据错误类型映射状态码。
var (
	ErrNotFound            = errors.New("resource not found")
	ErrAlreadyExists       = errors.New("resource already exists")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrValidation          = errors.New("validation error")
	ErrConflict            = errors.New("conflict")
	ErrInternal            = errors.New("internal error")
	ErrAccountFrozen       = errors.New("account is frozen")
	ErrAccountBanned       = errors.New("account is banned")
	ErrCreditRestricted    = errors.New("credit level is restricted for this action")
	ErrTooManyOpenPosts    = errors.New("too many open posts")
	ErrCannotApplyOwnPost  = errors.New("cannot apply to your own post")
	ErrAlreadyApplied      = errors.New("already applied to this post")
	ErrPostNotOpen         = errors.New("post is not open for applications")
	ErrPostExpired         = errors.New("post has expired")
	ErrAlreadyMatched      = errors.New("post already matched")
	ErrNotPostAuthor       = errors.New("only the post author can perform this action")
	ErrNotTaskParty        = errors.New("only task parties can perform this action")
	ErrNotHelper           = errors.New("only the helper can mark the task complete")
	ErrNotRequester        = errors.New("only the requester can confirm completion")
	ErrInvalidTaskStatus   = errors.New("task status does not allow this action")
	ErrAlreadyReviewed     = errors.New("already reviewed this task")
	ErrTaskNotCompleted    = errors.New("cannot review before the task is completed")
	ErrBothStartRequired   = errors.New("both parties must confirm start")
	ErrInvalidReviewTags   = errors.New("review tags do not match the score")
	ErrCannotDeleteUser    = errors.New("cannot delete user with active tasks")
	ErrDuplicateFavorite   = errors.New("post already favorited")
	ErrReportAlreadyOpen   = errors.New("an open report already exists for this target")
	ErrInvalidHandleAction = errors.New("invalid report handle action")

	ErrInvalidUsername    = errors.New("invalid username: 3-32 letters, digits or underscore")
	ErrInvalidPassword    = errors.New("invalid password: 6-64 characters")
	ErrInvalidRole        = errors.New("invalid role: must be admin or resident")
	ErrInvalidDisplayName = errors.New("invalid display name: 1-32 characters")
	ErrInvalidTitle       = errors.New("invalid title: 1-80 characters")
	ErrInvalidContent     = errors.New("invalid content: 1-4000 characters")
	ErrInvalidCategory    = errors.New("invalid category")
	ErrInvalidPostType    = errors.New("invalid post type: must be request or offer")
	ErrInvalidUrgency     = errors.New("invalid urgency")
	ErrInvalidStatus      = errors.New("invalid status")
	ErrInvalidScore       = errors.New("invalid score: 1-5")
	ErrInvalidPhone       = errors.New("invalid phone: 0-20 characters")
	ErrInvalidLocation    = errors.New("invalid building/unit/room: max 16 characters")
	ErrInvalidBio         = errors.New("invalid bio: max 200 characters")
	ErrInvalidComment     = errors.New("invalid comment: max 500 characters")
	ErrInvalidTimeWindow  = errors.New("invalid time window: end must be after start")
	ErrInvalidCreditDelta = errors.New("invalid credit delta: -20 to +20")
	ErrInvalidReward      = errors.New("invalid reward type")
	ErrInvalidUserStatus  = errors.New("invalid user status")
)

func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

func IsAlreadyExists(err error) bool { return errors.Is(err, ErrAlreadyExists) }

func IsUnauthorized(err error) bool { return errors.Is(err, ErrUnauthorized) }

func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden) ||
		errors.Is(err, ErrAccountFrozen) ||
		errors.Is(err, ErrAccountBanned) ||
		errors.Is(err, ErrCreditRestricted) ||
		errors.Is(err, ErrNotPostAuthor) ||
		errors.Is(err, ErrNotTaskParty) ||
		errors.Is(err, ErrNotHelper) ||
		errors.Is(err, ErrNotRequester)
}

func IsInvalidCredentials(err error) bool { return errors.Is(err, ErrInvalidCredentials) }

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict) ||
		errors.Is(err, ErrTooManyOpenPosts) ||
		errors.Is(err, ErrCannotApplyOwnPost) ||
		errors.Is(err, ErrAlreadyApplied) ||
		errors.Is(err, ErrPostNotOpen) ||
		errors.Is(err, ErrPostExpired) ||
		errors.Is(err, ErrAlreadyMatched) ||
		errors.Is(err, ErrInvalidTaskStatus) ||
		errors.Is(err, ErrAlreadyReviewed) ||
		errors.Is(err, ErrTaskNotCompleted) ||
		errors.Is(err, ErrBothStartRequired) ||
		errors.Is(err, ErrCannotDeleteUser) ||
		errors.Is(err, ErrDuplicateFavorite) ||
		errors.Is(err, ErrReportAlreadyOpen)
}

func IsValidation(err error) bool {
	switch {
	case errors.Is(err, ErrValidation),
		errors.Is(err, ErrInvalidUsername),
		errors.Is(err, ErrInvalidPassword),
		errors.Is(err, ErrInvalidRole),
		errors.Is(err, ErrInvalidDisplayName),
		errors.Is(err, ErrInvalidTitle),
		errors.Is(err, ErrInvalidContent),
		errors.Is(err, ErrInvalidCategory),
		errors.Is(err, ErrInvalidPostType),
		errors.Is(err, ErrInvalidUrgency),
		errors.Is(err, ErrInvalidStatus),
		errors.Is(err, ErrInvalidScore),
		errors.Is(err, ErrInvalidPhone),
		errors.Is(err, ErrInvalidLocation),
		errors.Is(err, ErrInvalidBio),
		errors.Is(err, ErrInvalidComment),
		errors.Is(err, ErrInvalidTimeWindow),
		errors.Is(err, ErrInvalidCreditDelta),
		errors.Is(err, ErrInvalidReward),
		errors.Is(err, ErrInvalidUserStatus),
		errors.Is(err, ErrInvalidReviewTags),
		errors.Is(err, ErrInvalidHandleAction):
		return true
	}
	return false
}
