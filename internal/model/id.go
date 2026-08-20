package model

// 资源 ID 与会话 Token 前缀。store 层生成 ID 时拼接此前缀，便于排查与日志检索。

const (
	UserIDPrefix         = "u_"
	PostIDPrefix         = "p_"
	ApplicationIDPrefix  = "a_"
	TaskIDPrefix         = "k_"
	ReviewIDPrefix       = "v_"
	MessageIDPrefix      = "m_"
	NotificationIDPrefix = "n_"
	ReportIDPrefix       = "r_"
	FavoriteIDPrefix     = "f_"
	CreditLogIDPrefix    = "c_"
	TokenPrefix          = "t_"
)
