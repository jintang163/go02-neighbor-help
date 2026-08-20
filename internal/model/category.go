package model

// Category 互助分类。
type Category string

const (
	CategoryGrocery   Category = "grocery"   // 买菜代购
	CategoryDelivery  Category = "delivery"  // 代取快递
	CategoryPet       Category = "pet"       // 照看宠物
	CategoryMoving    Category = "moving"    // 搬运重物
	CategoryRepair    Category = "repair"    // 上门修理
	CategoryEscort    Category = "escort"    // 陪同就医
	CategoryChildcare Category = "childcare" // 临时看护
	CategoryOther     Category = "other"     // 其他
)

// IsValid 分类是否合法。
func (c Category) IsValid() bool {
	switch c {
	case CategoryGrocery, CategoryDelivery, CategoryPet, CategoryMoving,
		CategoryRepair, CategoryEscort, CategoryChildcare, CategoryOther:
		return true
	}
	return false
}

// Label 中文标签。
func (c Category) Label() string {
	switch c {
	case CategoryGrocery:
		return "买菜代购"
	case CategoryDelivery:
		return "代取快递"
	case CategoryPet:
		return "照看宠物"
	case CategoryMoving:
		return "搬运重物"
	case CategoryRepair:
		return "上门修理"
	case CategoryEscort:
		return "陪同就医"
	case CategoryChildcare:
		return "临时看护"
	case CategoryOther:
		return "其他"
	default:
		return string(c)
	}
}

// AllCategories 全部合法分类（前端字典）。
func AllCategories() []CategoryInfo {
	cs := []Category{
		CategoryGrocery, CategoryDelivery, CategoryPet, CategoryMoving,
		CategoryRepair, CategoryEscort, CategoryChildcare, CategoryOther,
	}
	out := make([]CategoryInfo, 0, len(cs))
	for _, c := range cs {
		out = append(out, CategoryInfo{ID: c, Label: c.Label()})
	}
	return out
}

// CategoryInfo 分类字典项。
type CategoryInfo struct {
	ID    Category `json:"id"`
	Label string   `json:"label"`
}

// Urgency 紧急程度。
type Urgency string

const (
	UrgencyLow    Urgency = "low"
	UrgencyNormal Urgency = "normal"
	UrgencyHigh   Urgency = "high"
	UrgencyUrgent Urgency = "urgent"
)

func (u Urgency) IsValid() bool {
	switch u {
	case UrgencyLow, UrgencyNormal, UrgencyHigh, UrgencyUrgent:
		return true
	}
	return false
}

// Rank 排序权重，越大越靠前。
func (u Urgency) Rank() int {
	switch u {
	case UrgencyUrgent:
		return 4
	case UrgencyHigh:
		return 3
	case UrgencyNormal:
		return 2
	case UrgencyLow:
		return 1
	default:
		return 0
	}
}

// Label 中文标签。
func (u Urgency) Label() string {
	switch u {
	case UrgencyUrgent:
		return "紧急"
	case UrgencyHigh:
		return "较急"
	case UrgencyNormal:
		return "普通"
	case UrgencyLow:
		return "不急"
	default:
		return string(u)
	}
}

// RequiresMinCredit 发帖或报名该紧急程度所需最低信用分。
func (u Urgency) RequiresMinCredit() int {
	switch u {
	case UrgencyUrgent:
		return 50
	case UrgencyHigh:
		return 40
	default:
		return 0
	}
}

// RewardType 答谢方式（仅记录，不涉及资金清算）。
type RewardType string

const (
	RewardNone    RewardType = "none"
	RewardThanks  RewardType = "thanks"
	RewardPoints  RewardType = "points"
	RewardInKind  RewardType = "in_kind"
)

func (r RewardType) IsValid() bool {
	switch r {
	case RewardNone, RewardThanks, RewardPoints, RewardInKind, "":
		return true
	}
	return false
}

func (r RewardType) Normalize() RewardType {
	if r == "" {
		return RewardNone
	}
	return r
}
