package middleware

import (
	"context"
	"net/http"
	"strings"

	"go02-neighbor-help/internal/auth"
	"go02-neighbor-help/internal/model"
	"go02-neighbor-help/internal/respond"
	"go02-neighbor-help/internal/service"
)

const bearerPrefix = "Bearer "

func extractToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" || !strings.HasPrefix(h, bearerPrefix) {
		return ""
	}
	return strings.TrimSpace(h[len(bearerPrefix):])
}

func RequireAuth(sessions *auth.SessionManager, users UserLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			sess, err := sessions.Get(token)
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "会话无效或已过期，请重新登录")
				return
			}
			user, err := users.GetUserByID(r.Context(), sess.UserID)
			if err != nil {
				sessions.Invalidate(token)
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "用户不存在，请重新登录")
				return
			}
			if user.IsBanned() {
				sessions.Invalidate(token)
				respond.Error(w, http.StatusForbidden, "forbidden", "账号已被封禁")
				return
			}
			ctx := service.WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...model.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[model.UserRole]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := service.UserFromContext(r.Context())
			if !ok {
				respond.Error(w, http.StatusUnauthorized, "unauthorized", "未登录")
				return
			}
			if _, ok := allowed[user.Role]; !ok {
				respond.Error(w, http.StatusForbidden, "forbidden", "无权限执行此操作")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAdmin() func(http.Handler) http.Handler {
	return RequireRole(model.RoleAdmin)
}

type UserLookup interface {
	GetUserByID(ctx context.Context, id string) (model.User, error)
}
