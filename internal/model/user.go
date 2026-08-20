package model

import (
	"strings"
	"time"
	"unicode/utf8"
)

// UserRole 用户角色。
type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleResident UserRole = "resident"
)

func (r UserRole) IsValid() bool {
	return r == RoleAdmin || r == RoleResident
}

// UserStatus 账号状态。
type UserStatus string

const (
	UserActive  UserStatus = "active"
	UserFrozen  UserStatus = "frozen"
	UserBanned  UserStatus = "banned"
)

func (s UserStatus) IsValid() bool {
	return s == UserActive || s == UserFrozen || s == UserBanned || s == ""
}

func (s UserStatus) Normalize() UserStatus {
	if s == "" {
		return UserActive
	}
	return s
}

// CreditLevel 信用等级，由分数推导，不单独持久化权威值。
type CreditLevel string

const (
	CreditRestricted CreditLevel = "restricted"
	CreditNew        CreditLevel = "new"
	CreditNormal     CreditLevel = "normal"
	CreditTrusted    CreditLevel = "trusted"
	CreditExcellent  CreditLevel = "excellent"
)

const (
	CreditMin           = 0
	CreditMax           = 100
	CreditInitial       = 60
	CreditRestrictedMax = 39
	CreditNewMax        = 59
	CreditNormalMax     = 74
	CreditTrustedMax    = 89
)

// CreditLevelOf 由分数推导等级。
func CreditLevelOf(score int) CreditLevel {
	switch {
	case score <= CreditRestrictedMax:
		return CreditRestricted
	case score <= CreditNewMax:
		return CreditNew
	case score <= CreditNormalMax:
		return CreditNormal
	case score <= CreditTrustedMax:
		return CreditTrusted
	default:
		return CreditExcellent
	}
}

// ClampCredit 将分数限制在 [0,100]。
func ClampCredit(score int) int {
	if score < CreditMin {
		return CreditMin
	}
	if score > CreditMax {
		return CreditMax
	}
	return score
}

// Rank 列表加权：excellent=4 ... restricted=0。
func (l CreditLevel) Rank() int {
	switch l {
	case CreditExcellent:
		return 4
	case CreditTrusted:
		return 3
	case CreditNormal:
		return 2
	case CreditNew:
		return 1
	default:
		return 0
	}
}

// User 用户实体。口令仅存哈希与盐。
type User struct {
	ID            string     `json:"id"`
	Username      string     `json:"username"`
	PasswordHash  string     `json:"password_hash"`
	PasswordSalt  string     `json:"password_salt"`
	Iterations    int        `json:"iterations"`
	Role          UserRole   `json:"role"`
	Status        UserStatus `json:"status"`
	DisplayName   string     `json:"display_name"`
	Building      string     `json:"building,omitempty"`
	Unit          string     `json:"unit,omitempty"`
	Room          string     `json:"room,omitempty"`
	Phone         string     `json:"phone,omitempty"`
	Bio           string     `json:"bio,omitempty"`
	CreditScore   int        `json:"credit_score"`
	HelpCount     int        `json:"help_count"`
	RequestCount  int        `json:"request_count"`
	ReviewCount   int        `json:"review_count"`
	ReviewSum     int        `json:"review_sum"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
}

func (u User) IsAdmin() bool    { return u.Role == RoleAdmin }
func (u User) IsResident() bool { return u.Role == RoleResident }
func (u User) IsActive() bool   { return u.Status.Normalize() == UserActive }
func (u User) IsFrozen() bool   { return u.Status == UserFrozen }
func (u User) IsBanned() bool   { return u.Status == UserBanned }

func (u User) CreditLevel() CreditLevel { return CreditLevelOf(u.CreditScore) }

func (u User) AverageScore() float64 {
	if u.ReviewCount <= 0 {
		return 0
	}
	return float64(u.ReviewSum) / float64(u.ReviewCount)
}

func (u User) CanWrite() error {
	switch u.Status.Normalize() {
	case UserBanned:
		return ErrAccountBanned
	case UserFrozen:
		return ErrAccountFrozen
	}
	return nil
}

func (u User) LocationLabel() string {
	parts := make([]string, 0, 3)
	if u.Building != "" {
		parts = append(parts, u.Building+"栋")
	}
	if u.Unit != "" {
		parts = append(parts, u.Unit+"单元")
	}
	if u.Room != "" {
		parts = append(parts, u.Room)
	}
	return strings.Join(parts, " ")
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:           u.ID,
		Username:     u.Username,
		DisplayName:  u.DisplayName,
		Role:         u.Role,
		Status:       u.Status.Normalize(),
		Building:     u.Building,
		Unit:         u.Unit,
		Room:         u.Room,
		Bio:          u.Bio,
		CreditScore:  u.CreditScore,
		CreditLevel:  u.CreditLevel(),
		HelpCount:    u.HelpCount,
		RequestCount: u.RequestCount,
		ReviewCount:  u.ReviewCount,
		AverageScore: u.AverageScore(),
		CreatedAt:    u.CreatedAt,
	}
}

// PublicUser 对外资料（不含口令哈希）。
type PublicUser struct {
	ID           string      `json:"id"`
	Username     string      `json:"username"`
	DisplayName  string      `json:"display_name"`
	Role         UserRole    `json:"role"`
	Status       UserStatus  `json:"status"`
	Building     string      `json:"building,omitempty"`
	Unit         string      `json:"unit,omitempty"`
	Room         string      `json:"room,omitempty"`
	Bio          string      `json:"bio,omitempty"`
	CreditScore  int         `json:"credit_score"`
	CreditLevel  CreditLevel `json:"credit_level"`
	HelpCount    int         `json:"help_count"`
	RequestCount int         `json:"request_count"`
	ReviewCount  int         `json:"review_count"`
	AverageScore float64     `json:"average_score"`
	CreatedAt    time.Time   `json:"created_at"`
}

type UserInput struct {
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	Role        UserRole  `json:"role"`
	DisplayName string    `json:"display_name"`
	Building    string    `json:"building"`
	Unit        string    `json:"unit"`
	Room        string    `json:"room"`
	Phone       string    `json:"phone"`
	Bio         string    `json:"bio"`
}

func (in *UserInput) Normalize() {
	in.Username = strings.TrimSpace(in.Username)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Building = strings.TrimSpace(in.Building)
	in.Unit = strings.TrimSpace(in.Unit)
	in.Room = strings.TrimSpace(in.Room)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Bio = strings.TrimSpace(in.Bio)
	if in.Role == "" {
		in.Role = RoleResident
	}
}

func (in UserInput) Validate() error {
	in.Normalize()
	if !IsValidUsername(in.Username) {
		return ErrInvalidUsername
	}
	if !IsValidPassword(in.Password) {
		return ErrInvalidPassword
	}
	if !in.Role.IsValid() {
		return ErrInvalidRole
	}
	if err := ValidateDisplayName(in.DisplayName); err != nil {
		return err
	}
	if err := ValidateLocation(in.Building, in.Unit, in.Room); err != nil {
		return err
	}
	if utf8.RuneCountInString(in.Phone) > 20 {
		return ErrInvalidPhone
	}
	if utf8.RuneCountInString(in.Bio) > 200 {
		return ErrInvalidBio
	}
	return nil
}

type ProfileInput struct {
	DisplayName string `json:"display_name"`
	Building    string `json:"building"`
	Unit        string `json:"unit"`
	Room        string `json:"room"`
	Phone       string `json:"phone"`
	Bio         string `json:"bio"`
}

func (in *ProfileInput) Normalize() {
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Building = strings.TrimSpace(in.Building)
	in.Unit = strings.TrimSpace(in.Unit)
	in.Room = strings.TrimSpace(in.Room)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Bio = strings.TrimSpace(in.Bio)
}

func (in ProfileInput) Validate() error {
	in.Normalize()
	if err := ValidateDisplayName(in.DisplayName); err != nil {
		return err
	}
	if err := ValidateLocation(in.Building, in.Unit, in.Room); err != nil {
		return err
	}
	if utf8.RuneCountInString(in.Phone) > 20 {
		return ErrInvalidPhone
	}
	if utf8.RuneCountInString(in.Bio) > 200 {
		return ErrInvalidBio
	}
	return nil
}

func ValidateDisplayName(s string) error {
	n := utf8.RuneCountInString(strings.TrimSpace(s))
	if n < 1 || n > 32 {
		return ErrInvalidDisplayName
	}
	return nil
}

func ValidateLocation(building, unit, room string) error {
	if utf8.RuneCountInString(building) > 16 || utf8.RuneCountInString(unit) > 16 || utf8.RuneCountInString(room) > 16 {
		return ErrInvalidLocation
	}
	return nil
}

func IsValidUsername(s string) bool {
	if len(s) < 3 || len(s) > 32 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
	}
	return true
}

func IsValidPassword(s string) bool {
	return len(s) >= 6 && len(s) <= 64
}

type UserFilter struct {
	Role   UserRole
	Status UserStatus
	Query  string
}
